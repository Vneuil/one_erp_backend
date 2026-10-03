package tenant

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/config"
	tenantDomain "github.com/divinecoid/one-backend/internal/modules/tenant/domain"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	evictScanInterval = 5 * time.Minute
	idleEvictAfter    = 30 * time.Minute
)

type pooledConn struct {
	db       *gorm.DB
	lastUsed time.Time
}

// Manager holds one lazily-created *gorm.DB connection pool per tenant
// (company) database, keyed by company ID. Idle pools are evicted
// periodically to bound the number of open connections.
type Manager struct {
	dbCfg *config.DatabaseConfig
	repo  tenantDomain.TenantDatabaseRepository

	mu    sync.Mutex
	pools map[uuid.UUID]*pooledConn

	stopCh chan struct{}
}

func NewManager(dbCfg *config.DatabaseConfig, repo tenantDomain.TenantDatabaseRepository) *Manager {
	m := &Manager{
		dbCfg:  dbCfg,
		repo:   repo,
		pools:  make(map[uuid.UUID]*pooledConn),
		stopCh: make(chan struct{}),
	}
	go m.evictLoop()
	return m
}

// GetDB returns (lazily creating if needed) the connection pool for the
// given company's tenant database.
func (m *Manager) GetDB(ctx context.Context, companyID uuid.UUID) (*gorm.DB, error) {
	m.mu.Lock()
	if pc, ok := m.pools[companyID]; ok {
		pc.lastUsed = time.Now()
		m.mu.Unlock()
		return pc.db, nil
	}
	m.mu.Unlock()

	record, err := m.repo.GetByCompanyID(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to look up tenant database for company %s: %w", companyID, err)
	}
	if record == nil {
		return nil, fmt.Errorf("no tenant database provisioned for company %s", companyID)
	}

	db, err := gorm.Open(postgres.Open(m.dbCfg.DSNForDatabase(record.DatabaseName)), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open tenant database connection for company %s: %w", companyID, err)
	}

	m.mu.Lock()
	if existing, ok := m.pools[companyID]; ok {
		// another goroutine won the race - close the connection we just opened
		m.mu.Unlock()
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		existing.lastUsed = time.Now()
		return existing.db, nil
	}
	m.pools[companyID] = &pooledConn{db: db, lastUsed: time.Now()}
	m.mu.Unlock()

	return db, nil
}

func (m *Manager) evictLoop() {
	ticker := time.NewTicker(evictScanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.evictIdle()
		case <-m.stopCh:
			return
		}
	}
}

func (m *Manager) evictIdle() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for companyID, pc := range m.pools {
		if now.Sub(pc.lastUsed) > idleEvictAfter {
			if sqlDB, err := pc.db.DB(); err == nil {
				_ = sqlDB.Close()
			}
			delete(m.pools, companyID)
			slog.Info("evicted idle tenant database pool", "companyId", companyID)
		}
	}
}

// Close closes all pooled tenant connections and stops the eviction loop.
// Intended to be wired into graceful shutdown.
func (m *Manager) Close() {
	close(m.stopCh)
	m.mu.Lock()
	defer m.mu.Unlock()
	for companyID, pc := range m.pools {
		if sqlDB, err := pc.db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		delete(m.pools, companyID)
	}
}
