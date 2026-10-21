package main

import (
	"context"
	"go-agent/api"
	"go-agent/config"
	"go-agent/flow"
	"go-agent/flow/collaborative_writing"
	"go-agent/model/chat_model"
	"go-agent/rag/article"
	"go-agent/rag/rag_flow"
	"go-agent/rag/rag_tools/db"
	"go-agent/rag/rag_tools/indexer"
	"go-agent/rag/rag_tools/retriever"
	"go-agent/scheduler"
	"go-agent/tool/blog_tools"
	"go-agent/tool/document"
	"go-agent/tool/memory"
	"go-agent/tool/skill"
	"go-agent/tool/storage"
	"go-agent/tool/trace"
	"log"
	"time"

	"github.com/cloudwego/eino-ext/devops"
)

func main() {
	var err error
	ctx := context.Background()

	config.Cfg, err = config.LoadConfig()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	err = devops.Init(ctx)
	if err != nil {
		log.Printf("[eino dev] init failed, err=%v", err)
	}

	dependencyCtx, cancelDependencyProbe := context.WithTimeout(ctx, 3*time.Second)
	err = storage.InitRedis(dependencyCtx)
	cancelDependencyProbe()
	if err != nil {
		log.Printf("warning: Redis init failed, using in-memory mode: %v", err)
	}
	defer storage.CloseRedis()

	vectorReady := false
	if config.Cfg.VectorDBType == "milvus" {
		milvusCtx, cancelMilvusProbe := context.WithTimeout(ctx, 3*time.Second)
		db.Milvus, err = db.NewMilvus(milvusCtx)
		cancelMilvusProbe()
		if err != nil {
			log.Printf("warning: Milvus init failed, RAG index/retrieval unavailable: %v", err)
		} else {
			vectorReady = true
			defer db.Milvus.Close()
		}
	} else if config.Cfg.VectorDBType == "es" {
		vectorReady = true
	}

	indexer.NewIndexer()
	retriever.NewRetriever()

	document.Parser, err = document.NewParser(ctx)
	if err != nil {
		log.Fatalf("parser init fail: %v", err)
	}

	document.Loader, err = document.NewLoader(ctx)
	if err != nil {
		log.Fatalf("loader init fail: %v", err)
	}

	document.Splitter, err = document.NewSplitter(ctx)
	if err != nil {
		log.Fatalf("splitter init fail: %v", err)
	}
	article.SetDefaultIndexService(article.NewDefaultIndexService(document.Splitter))

	err = trace.NewLangSmith()
	if err != nil {
		log.Printf("warning: LangSmith init failed, tracing skipped: %v", err)
	}

	closeCoze, err := trace.NewCozeLoop(ctx)
	if err != nil {
		log.Printf("warning: CozeLoop init failed, tracing skipped: %v", err)
		closeCoze = func() {}
	}
	defer closeCoze()

	if vectorReady {
		err = rag_flow.InitIndexingGraph(ctx)
		if err != nil {
			vectorReady = false
			log.Printf("warning: IndexingGraph init failed, RAG document indexing unavailable: %v", err)
		} else {
			log.Println("IndexingGraph compiled and cached")
		}
	} else {
		log.Println("warning: vector retrieval dependency unavailable, skipping IndexingGraph init")
	}

	var memStore memory.Store = memory.NewMemoryStore()
	if redisClient, redisClientErr := storage.GetRedisClient(); redisClientErr == nil {
		memStore = memory.NewRedisStore(
			redisClient,
			config.Cfg.MemoryConf.KeyPrefix,
			config.Cfg.MemoryConf.HistoryTTL,
			config.Cfg.MemoryConf.SummaryTTL,
		)
		log.Println("Redis session memory enabled")
	} else {
		log.Printf("warning: Redis session memory unavailable, using in-memory store: %v", redisClientErr)
	}
	taskModel, err := chat_model.GetChatModel(ctx, config.Cfg.ChatModelType)
	if err != nil {
		log.Fatalf("task model init fail: %v", err)
	}

	// 初始化长期记忆（跨会话 User Memory）。MySQL 不可用时降级为 Noop，系统退回纯 session 模式。
	var userMemStore memory.UserMemoryStore = memory.NoopUserMemoryStore{}
	var skillStore skill.SkillStore = skill.NoopSkillStore{}
	mysqlCtx, cancelMySQLProbe := context.WithTimeout(ctx, 5*time.Second)
	mysqlDB, mysqlErr := storage.InitMySQL(mysqlCtx)
	cancelMySQLProbe()
	if mysqlErr != nil {
		log.Printf("warning: MySQL unavailable, long-term memory and skills disabled: %v", mysqlErr)
	} else {
		if store, storeErr := memory.NewMySQLUserMemoryStore(mysqlDB); storeErr != nil {
			log.Printf("warning: user memory store init failed, long-term memory disabled: %v", storeErr)
		} else {
			userMemStore = store
			flow.SetMemoryPipeline(memory.NewFactExtractor(taskModel, 0), userMemStore)
			log.Println("Long-term memory (User Memory) initialized")
		}

		if store, storeErr := skill.NewMySQLSkillStore(mysqlDB); storeErr != nil {
			log.Printf("warning: skill store init failed, skill learning disabled: %v", storeErr)
		} else {
			skillStore = store
			flow.SetSkillPipeline(skill.NewSkillExtractor(taskModel), skillStore)
			blog_tools.SetSkillStore(skillStore)
			log.Println("Skill learning initialized")
		}
	}

	if err := flow.InitBlogAssistant(ctx, taskModel); err != nil {
		log.Printf("warning: BlogAssistant init failed, blog domain tools unavailable: %v", err)
	} else {
		log.Println("BlogAssistant compiled and cached")
	}
	if err := flow.InitWritingFlow(taskModel); err != nil {
		log.Printf("warning: WritingFlow init failed, AI writing unavailable: %v", err)
	} else {
		log.Println("WritingFlow initialized")
	}
	if err := flow.InitContentReviewAgent(taskModel); err != nil {
		log.Printf("warning: ContentReviewAgent init failed, content review unavailable: %v", err)
	} else {
		log.Println("ContentReviewAgent initialized")
	}
	if err := flow.InitInsightFlows(taskModel); err != nil {
		log.Printf("warning: InsightFlows init failed, summary/comment analytics unavailable: %v", err)
	} else {
		log.Println("InsightFlows initialized")
	}
	ragEnabled := config.Cfg.ESConf.Index != "" || config.Cfg.MilvusConf.CollectionName != ""
	if ragEnabled {
		err = flow.InitRAGChatFlow(ctx, memStore, taskModel)
		if err != nil {
			log.Printf("warning: RAGChatFlow init failed, RAG chat unavailable: %v", err)
		} else {
			log.Println("RAGChatFlow compiled and cached")
		}
	} else {
		log.Println("warning: hybrid retrieval is not configured, skipping RAGChatFlow init")
	}
	if err := flow.InitUnifiedAssistant(taskModel); err != nil {
		log.Printf("warning: UnifiedAssistant init failed: %v", err)
	} else {
		log.Println("UnifiedAssistant initialized")
	}

	// 初始化统一对话流（合并 RAG 和工具调用）。检索器本身是惰性初始化，
	// 因此即使当前没有配置向量库，也可以先提供 blog_tool 分支。
	if err := flow.InitUnifiedChatFlow(ctx, memStore, taskModel); err != nil {
		log.Printf("warning: UnifiedChatFlow init failed: %v", err)
	} else {
		log.Println("UnifiedChatFlow compiled and cached (RAG + Tool merged)")
	}

	// 初始化多Agent协作写作系统
	if err := collaborative_writing.InitCollaborativeWritingGraph(ctx); err != nil {
		log.Printf("warning: CollaborativeWriting init failed: %v", err)
	} else {
		log.Println("CollaborativeWriting initialized (Multi-Agent System)")
	}

	// 初始化自动发布定时任务
	if config.Cfg.AutoPublish.Enabled {
		scheduler := scheduler.NewAutoPublishScheduler()
		dailyTime := "09:00" // 默认每天9点
		if config.Cfg.AutoPublish.Cron != "" {
			// 从cron表达式提取时间 (例如 "0 9 * * *" -> "09:00")
			// 简化处理：假设格式为 "分 时 * * *"
			dailyTime = config.Cfg.AutoPublish.Cron
		}
		if err := scheduler.Start(dailyTime); err != nil {
			log.Printf("warning: AutoPublish scheduler init failed: %v", err)
		} else {
			log.Printf("AutoPublish scheduler started (daily at %s)", dailyTime)
		}
	} else {
		log.Println("AutoPublish scheduler disabled")
	}

	// 初始化技能淘汰定时任务
	if curator, err := scheduler.NewSkillCurator(skillStore, "02:00", 3, 60*24*time.Hour); err == nil && curator != nil {
		if err := curator.Start("02:00"); err != nil {
			log.Printf("warning: SkillCurator init failed: %v", err)
		} else {
			log.Println("SkillCurator started (daily at 02:00)")
		}
	}

	api.Run()
}
