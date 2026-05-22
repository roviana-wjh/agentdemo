package main

import (
	"context"
	"go-agent/api"
	"go-agent/config"
	"go-agent/flow"
	"go-agent/model/chat_model"
	"go-agent/rag/rag_flow"
	"go-agent/rag/rag_tools/db"
	"go-agent/rag/rag_tools/indexer"
	"go-agent/rag/rag_tools/retriever"
	"go-agent/tool/document"
	"go-agent/tool/memory"
	"go-agent/tool/sql_tools"
	"go-agent/tool/storage"
	"go-agent/tool/trace"
	"log"
	"time"

	"github.com/cloudwego/eino-ext/devops"
)

func main() {
	var err error
	ctx := context.Background()

	// 初始化config
	config.Cfg, err = config.LoadConfig()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	err = devops.Init(ctx)
	if err != nil {
		log.Printf("[eino dev] init failed, err=%v", err)
	}

	// 初始化Redis
	dependencyCtx, cancelDependencyProbe := context.WithTimeout(ctx, 3*time.Second)
	err = storage.InitRedis(dependencyCtx)
	cancelDependencyProbe()
	if err != nil {
		log.Printf("警告: Redis 初始化失败，将使用内存模式: %v", err)
	}
	defer storage.CloseRedis()

	// 初始化数据库
	vectorReady := false
	if config.Cfg.VectorDBType == "milvus" {
		milvusCtx, cancelMilvusProbe := context.WithTimeout(ctx, 3*time.Second)
		db.Milvus, err = db.NewMilvus(milvusCtx)
		cancelMilvusProbe()
		if err != nil {
			log.Printf("警告: Milvus 初始化失败，RAG 索引/检索能力暂不可用: %v", err)
		} else {
			vectorReady = true
			defer db.Milvus.Close()
		}
	} else if config.Cfg.VectorDBType == "es" {
		vectorReady = true
	}

	// 初始化检索器
	indexer.NewIndexer()

	// 初始化召回器
	retriever.NewRetriever()

	// 初始化解析器
	document.Parser, err = document.NewParser(ctx)
	if err != nil {
		log.Fatalf("parser init fail: %v", err)
	}

	// 初始化载入器
	document.Loader, err = document.NewLoader(ctx)
	if err != nil {
		log.Fatalf("loader init fail: %v", err)
	}

	// 初始化切分器
	document.Splitter, err = document.NewSplitter(ctx)
	if err != nil {
		log.Fatalf("splitter init fail: %v", err)
	}

	// 初始化langsmith
	err = trace.NewLangSmith()
	if err != nil {
		log.Printf("警告: LangSmith 初始化失败，已跳过链路追踪: %v", err)
	}

	// 初始化 CozeLoop
	closeCoze, err := trace.NewCozeLoop(ctx)
	if err != nil {
		log.Printf("警告: CozeLoop 初始化失败，已跳过链路追踪: %v", err)
		closeCoze = func() {}
	}
	defer closeCoze()

	// 初始化MCP
	mcpReady := true
	if config.Cfg.MySQLConf.Username == "" || config.Cfg.MySQLConf.Database == "" {
		mcpReady = false
		log.Println("警告: MySQL 配置不完整，跳过 MCP 工具初始化")
	} else {
		mcpCtx, cancelMCPProbe := context.WithTimeout(ctx, 8*time.Second)
		err = sql_tools.InitMCPTools(mcpCtx)
		cancelMCPProbe()
		if err != nil {
			mcpReady = false
			log.Printf("警告: MCP 工具初始化失败，总控图 SQL 能力暂不可用: %v", err)
		} else {
			log.Println("MCP 工具连接已建立")
		}
	}

	// 预编译索引图
	if vectorReady {
		err = rag_flow.InitIndexingGraph(ctx)
		if err != nil {
			vectorReady = false
			log.Printf("警告: IndexingGraph 初始化失败，RAG 文档入库暂不可用: %v", err)
		} else {
			log.Println("IndexingGraph 已编译缓存")
		}
	} else {
		log.Println("警告: 向量检索依赖未就绪，跳过 IndexingGraph 初始化")
	}

	// 预编译RAG对话图
	memStore := memory.NewMemoryStore()
	taskModel, err := chat_model.GetChatModel(ctx, config.Cfg.ChatModelType)
	if err != nil {
		log.Fatalf("task model init fail: %v", err)
	}
	if vectorReady {
		err = flow.InitRAGChatFlow(ctx, memStore, taskModel)
		if err != nil {
			log.Printf("警告: RAGChatFlow 初始化失败，RAG 问答暂不可用: %v", err)
		} else {
			log.Println("RAGChatFlow 已编译缓存")
		}
	} else {
		log.Println("警告: 向量检索依赖未就绪，跳过 RAGChatFlow 初始化")
	}

	// 预编译全局图（使用Redis缓存）
	if mcpReady {
		checkPointStore := storage.NewRedisCheckPointStore()
		err = flow.InitFinalGraph(ctx, checkPointStore)
		if err != nil {
			log.Printf("警告: FinalGraph 初始化失败，总控图暂不可用: %v", err)
		} else {
			log.Println("FinalGraph 已编译缓存")
		}
	} else {
		log.Println("警告: MCP 未就绪，跳过 FinalGraph 初始化")
	}

	api.Run()
}
