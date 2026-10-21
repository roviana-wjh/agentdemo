package scheduler

import (
	"context"
	"log"
	"time"

	"go-agent/tool/skill"
	"go-agent/tool/trace"
)

// SkillCurator periodically evaluates and prunes unused skills using LRU/LFU strategies.
type SkillCurator struct {
	store              skill.SkillStore
	interval           time.Duration
	stopChan           chan struct{}
	minUsageThreshold  int           // LFU: minimum usage count threshold
	maxUnusedDuration  time.Duration // LRU: maximum time since last use
}

// NewSkillCurator creates a curator that runs cleanup tasks daily using LRU/LFU.
// minUsageThreshold: skills with usage < threshold and age > 30 days are deleted (LFU).
// maxUnusedDuration: skills unused for longer than this duration are deleted (LRU).
func NewSkillCurator(store skill.SkillStore, dailyTime string, minUsageThreshold int, maxUnusedDuration time.Duration) (*SkillCurator, error) {
	if store == nil {
		return nil, nil // graceful degradation: no-op if store unavailable
	}

	if minUsageThreshold <= 0 {
		minUsageThreshold = 3 // default: need at least 3 uses to survive
	}
	if maxUnusedDuration <= 0 {
		maxUnusedDuration = 60 * 24 * time.Hour // default: 60 days unused
	}

	return &SkillCurator{
		store:             store,
		stopChan:          make(chan struct{}),
		minUsageThreshold: minUsageThreshold,
		maxUnusedDuration: maxUnusedDuration,
	}, nil
}

// Start begins the daily cleanup task at the specified time (e.g., "02:00").
func (c *SkillCurator) Start(dailyTime string) error {
	if c == nil {
		return nil
	}

	targetTime, err := time.Parse("15:04", dailyTime)
	if err != nil {
		return err
	}

	go c.runDailyCleanup(targetTime.Hour(), targetTime.Minute())
	return nil
}

// Stop halts the curator.
func (c *SkillCurator) Stop() {
	if c == nil {
		return
	}
	close(c.stopChan)
}

func (c *SkillCurator) runDailyCleanup(hour, minute int) {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

		if next.Before(now) {
			next = next.Add(24 * time.Hour)
		}

		waitDuration := next.Sub(now)
		log.Printf("[SkillCurator] Next cleanup: %s", next.Format("2006-01-02 15:04:05"))

		select {
		case <-time.After(waitDuration):
			c.executeCleanup()
		case <-c.stopChan:
			log.Println("[SkillCurator] Stopped")
			return
		}
	}
}

func (c *SkillCurator) executeCleanup() {
	ctx := context.Background()

	log.Printf("[SkillCurator] Starting skill cleanup task (threshold=%d, unused_days=%d)",
		c.minUsageThreshold, int(c.maxUnusedDuration.Hours()/24))

	skills, err := c.store.ListAll(ctx)
	if err != nil {
		log.Printf("[SkillCurator] List skills failed: %v", err)
		return
	}

	log.Printf("[SkillCurator] Evaluating %d skills", len(skills))

	var lfuCount, lruCount int
	var allDeleted []string

	for _, s := range skills {
		// Rule 1 (LFU): Delete low-usage skills that are old enough
		if s.UsageCount < c.minUsageThreshold && time.Since(s.CreatedAt) > 30*24*time.Hour {
			if err := c.store.DeleteByName(ctx, s.Name); err != nil {
				log.Printf("[SkillCurator] Delete low-usage '%s' failed: %v", s.Name, err)
			} else {
				log.Printf("[SkillCurator] Deleted low-usage skill '%s' (usage: %d, age: %d days)",
					s.Name, s.UsageCount, int(time.Since(s.CreatedAt).Hours()/24))
				lfuCount++
				allDeleted = append(allDeleted, s.Name)
			}
			continue
		}

		// Rule 2 (LRU): Delete skills not used for a long time
		if !s.LastUsedAt.IsZero() && time.Since(s.LastUsedAt) > c.maxUnusedDuration {
			if err := c.store.DeleteByName(ctx, s.Name); err != nil {
				log.Printf("[SkillCurator] Delete stale '%s' failed: %v", s.Name, err)
			} else {
				log.Printf("[SkillCurator] Deleted stale skill '%s' (unused for %d days)",
					s.Name, int(time.Since(s.LastUsedAt).Hours()/24))
				lruCount++
				allDeleted = append(allDeleted, s.Name)
			}
			continue
		}

		// Fallback for skills never used (LastUsedAt is zero)
		if s.LastUsedAt.IsZero() && s.UsageCount == 0 && time.Since(s.CreatedAt) > 30*24*time.Hour {
			if err := c.store.DeleteByName(ctx, s.Name); err != nil {
				log.Printf("[SkillCurator] Delete unused '%s' failed: %v", s.Name, err)
			} else {
				log.Printf("[SkillCurator] Deleted never-used skill '%s' (age: %d days)",
					s.Name, int(time.Since(s.CreatedAt).Hours()/24))
				lruCount++
				allDeleted = append(allDeleted, s.Name)
			}
		}
	}

	// Log structured event for aggregation
	trace.LogCuratorCleanup(lfuCount, lruCount, len(skills), allDeleted)

	log.Printf("[SkillCurator] Cleanup complete: %d LFU deleted, %d LRU deleted, total evaluated: %d",
		lfuCount, lruCount, len(skills))
}
