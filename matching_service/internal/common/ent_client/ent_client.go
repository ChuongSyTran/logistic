package entclient

import (
	"context"
	"errors"
	"fmt"
	"log"
	"matching_service/ent"
	"matching_service/internal/conf"

	_ "matching_service/ent/runtime"

	_ "github.com/lib/pq"
)

var (
	ErrDBMasterConnect = errors.New("failed to open database master connection")
	ErrDBSlaveConnect  = errors.New("failed to open database slave connection")
	ErrDBPing          = errors.New("database is unreachable (ping failed)")
)

func NewConnection(masterCfg *conf.MasterDatabaseConfig, slaveCfg *conf.SlaveDatabaseConfig) (*ent.Client, *ent.Client, error) {
	if masterCfg == nil || slaveCfg == nil {
		log.Fatalf("[SYSTEM] failed to read postgres's config")
	}

	masterClient, err := ent.Open(masterCfg.Driver, masterCfg.GetDataSource())
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrDBMasterConnect, err)
	}

	slaveClient, err := ent.Open(slaveCfg.Driver, slaveCfg.GetDataSource())
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrDBSlaveConnect, err)
	}

	_, err = masterClient.QueryContext(context.Background(), "SELECT 1")
	if err != nil {
		log.Fatalf("[ENT] failed connection to postgres: %v", err)
	}

	if err := masterClient.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	return masterClient, slaveClient, nil
}