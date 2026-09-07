package entclient

import (
	"auth_service/internal/conf"
	"context"
	"errors"
	"fmt"
	"log"

	"auth_service/ent"
	_ "auth_service/ent/runtime"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

var (
	ErrDBConnect = errors.New("failed to open database connection")
	ErrDBPing    = errors.New("database is unreachable (ping failed)")
)

func NewConnection(dbCfg conf.DatabaseConfig) (*ent.Client, error) {
	client, err := ent.Open(dbCfg.Driver, dbCfg.GetDataSource())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBConnect, err)
	}

	_, err = client.QueryContext(context.Background(), "SELECT 1")
	if err != nil {
		log.Fatalf("[ENT] failed connection to postgres: %v", err)
	}

	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	return client, nil
}
