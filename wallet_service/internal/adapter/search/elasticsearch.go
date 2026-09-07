package search

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/google/uuid"
)

type WalletDocument struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	UserType  uint8     `json:"user_type"`
	Balance   int64     `json:"balance"`
	Currency  string    `json:"currency"`
	Status    uint8     `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TransactionDocument struct {
	ID              string    `json:"id"`
	WalletID        string    `json:"wallet_id"`
	Amount          int64     `json:"amount"`
	TransactionType uint8     `json:"transaction_type"`
	ReferenceID     string    `json:"reference_id"`
	Description     string    `json:"description"`
	Status          uint8     `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

type SearchWalletParams struct {
	Query    string
	UserType *uint8
	Status   *uint8
	Page     int
	PageSize int
}

type SearchTransactionParams struct {
	Query           string
	WalletID        *uuid.UUID
	TransactionType *uint8
	Status          *uint8
	From            *time.Time
	To              *time.Time
	Page            int
	PageSize        int
}

type SearchResult[T any] struct {
	Hits     []T   `json:"hits"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

type WalletSearchEngine interface {
	IndexWallet(ctx context.Context, doc *WalletDocument) error
	IndexTransaction(ctx context.Context, doc *TransactionDocument) error

	SearchWallets(ctx context.Context, params *SearchWalletParams) (*SearchResult[WalletDocument], error)
	SearchTransactions(ctx context.Context, params *SearchTransactionParams) (*SearchResult[TransactionDocument], error)

	EnsureIndices(ctx context.Context) error
}

const (
	WalletIndex      = "wallets"
	TransactionIndex = "transactions"
)

type elasticSearchEngine struct {
	client *elasticsearch.Client
}

func NewElasticSearchEngine(addresses []string, username, password string) (WalletSearchEngine, error) {
	cfg := elasticsearch.Config{
		Addresses: addresses,
		Username:  username,
		Password:  password,
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	res, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("elasticsearch connection failed: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch connection failed: %s", res.String())
	}

	log.Printf("[ES] Connected to Elasticsearch cluster")
	return &elasticSearchEngine{client: client}, nil
}

func (e *elasticSearchEngine) EnsureIndices(ctx context.Context) error {
	walletMapping := `{
		"mappings": {
			"properties": {
				"id":         { "type": "keyword" },
				"user_id":    { "type": "keyword" },
				"user_type":  { "type": "byte" },
				"balance":    { "type": "long" },
				"currency":   { "type": "keyword" },
				"status":     { "type": "byte" },
				"created_at": { "type": "date" },
				"updated_at": { "type": "date" }
			}
		}
	}`

	transactionMapping := `{
		"mappings": {
			"properties": {
				"id":               { "type": "keyword" },
				"wallet_id":        { "type": "keyword" },
				"amount":           { "type": "long" },
				"transaction_type": { "type": "byte" },
				"reference_id":     { "type": "text", "fields": { "keyword": { "type": "keyword" } } },
				"description":      { "type": "text", "analyzer": "standard" },
				"status":           { "type": "byte" },
				"created_at":       { "type": "date" }
			}
		}
	}`

	if err := e.createIndexIfNotExists(ctx, WalletIndex, walletMapping); err != nil {
		return err
	}
	return e.createIndexIfNotExists(ctx, TransactionIndex, transactionMapping)
}

func (e *elasticSearchEngine) createIndexIfNotExists(ctx context.Context, index, mapping string) error {
	res, err := e.client.Indices.Exists([]string{index}, e.client.Indices.Exists.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to check index %s: %w", index, err)
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		log.Printf("[ES] Index '%s' already exists", index)
		return nil
	}

	res, err = e.client.Indices.Create(
		index,
		e.client.Indices.Create.WithBody(strings.NewReader(mapping)),
		e.client.Indices.Create.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to create index %s: %w", index, err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error creating index %s: %s", index, res.String())
	}

	log.Printf("[ES] Created index '%s'", index)
	return nil
}

func (e *elasticSearchEngine) IndexWallet(ctx context.Context, doc *WalletDocument) error {
	return e.indexDocument(ctx, WalletIndex, doc.ID, doc)
}

func (e *elasticSearchEngine) IndexTransaction(ctx context.Context, doc *TransactionDocument) error {
	return e.indexDocument(ctx, TransactionIndex, doc.ID, doc)
}

func (e *elasticSearchEngine) indexDocument(ctx context.Context, index, id string, doc any) error {
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	req := esapi.IndexRequest{
		Index:      index,
		DocumentID: id,
		Body:       strings.NewReader(string(data)),
		Refresh:    "false",
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error indexing document [%s/%s]: %s", index, id, res.String())
	}

	return nil
}

func (e *elasticSearchEngine) SearchWallets(ctx context.Context, params *SearchWalletParams) (*SearchResult[WalletDocument], error) {
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	query := buildWalletQuery(params)
	return doSearch[WalletDocument](ctx, e.client, WalletIndex, query, params.Page, params.PageSize)
}

func buildWalletQuery(params *SearchWalletParams) map[string]any {
	must := make([]map[string]any, 0)

	if params.Query != "" {
		must = append(must, map[string]any{
			"multi_match": map[string]any{
				"query":  params.Query,
				"fields": []string{"user_id", "currency"},
				"type":   "phrase_prefix",
			},
		})
	}
	if params.UserType != nil {
		must = append(must, map[string]any{
			"term": map[string]any{"user_type": *params.UserType},
		})
	}
	if params.Status != nil {
		must = append(must, map[string]any{
			"term": map[string]any{"status": *params.Status},
		})
	}

	if len(must) == 0 {
		return map[string]any{"query": map[string]any{"match_all": map[string]any{}}}
	}
	return map[string]any{"query": map[string]any{"bool": map[string]any{"must": must}}}
}

func (e *elasticSearchEngine) SearchTransactions(ctx context.Context, params *SearchTransactionParams) (*SearchResult[TransactionDocument], error) {
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	query := buildTransactionQuery(params)
	return doSearch[TransactionDocument](ctx, e.client, TransactionIndex, query, params.Page, params.PageSize)
}

func buildTransactionQuery(params *SearchTransactionParams) map[string]any {
	must := make([]map[string]any, 0)

	if params.Query != "" {
		must = append(must, map[string]any{
			"multi_match": map[string]any{
				"query":  params.Query,
				"fields": []string{"reference_id", "description"},
				"type":   "phrase_prefix",
			},
		})
	}
	if params.WalletID != nil {
		must = append(must, map[string]any{
			"term": map[string]any{"wallet_id": params.WalletID.String()},
		})
	}
	if params.TransactionType != nil {
		must = append(must, map[string]any{
			"term": map[string]any{"transaction_type": *params.TransactionType},
		})
	}
	if params.Status != nil {
		must = append(must, map[string]any{
			"term": map[string]any{"status": *params.Status},
		})
	}

	if params.From != nil || params.To != nil {
		rangeFilter := map[string]any{}
		if params.From != nil {
			rangeFilter["gte"] = params.From.Format(time.RFC3339)
		}
		if params.To != nil {
			rangeFilter["lte"] = params.To.Format(time.RFC3339)
		}
		must = append(must, map[string]any{
			"range": map[string]any{"created_at": rangeFilter},
		})
	}

	if len(must) == 0 {
		return map[string]any{"query": map[string]any{"match_all": map[string]any{}}}
	}
	return map[string]any{"query": map[string]any{"bool": map[string]any{"must": must}}}
}

func doSearch[T any](ctx context.Context, client *elasticsearch.Client, index string, query map[string]any, page, pageSize int) (*SearchResult[T], error) {
	from := (page - 1) * pageSize
	query["from"] = from
	query["size"] = pageSize
	query["sort"] = []map[string]any{
		{"created_at": map[string]any{"order": "desc"}},
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	res, err := client.Search(
		client.Search.WithContext(ctx),
		client.Search.WithIndex(index),
		client.Search.WithBody(strings.NewReader(string(body))),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("search error on index %s: %s", index, res.String())
	}

	var esResponse struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source T `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	hits := make([]T, 0, len(esResponse.Hits.Hits))
	for _, h := range esResponse.Hits.Hits {
		hits = append(hits, h.Source)
	}

	return &SearchResult[T]{
		Hits:     hits,
		Total:    esResponse.Hits.Total.Value,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
