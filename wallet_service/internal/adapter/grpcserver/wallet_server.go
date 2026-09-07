package grpcserver

import (
	"context"

	"github.com/google/uuid"
	pb "github.com/logistic/api/logistic/wallet_service/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"wallet_service/internal/adapter/search"
	"wallet_service/internal/app"
	"wallet_service/internal/mapper"
)

type WalletServer struct {
	pb.UnimplementedWalletServiceServer
	useCase  app.WalletUseCase
	esEngine search.WalletSearchEngine
	mapper   mapper.WalletMapper
}

func NewWalletServer(useCase app.WalletUseCase, esEngine search.WalletSearchEngine, m mapper.WalletMapper) *WalletServer {
	return &WalletServer{
		useCase:  useCase,
		esEngine: esEngine,
		mapper:   m,
	}
}

func (c *WalletServer) GetBalance(ctx context.Context, req *pb.GetBalanceReq) (*pb.GetBalanceRes, error) {
	userID, err := uuid.FromBytes(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id format: %v", err)
	}

	wallet, err := c.useCase.GetBalance(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &pb.GetBalanceRes{
		Balance:  wallet.Balance,
		Currency: wallet.Currency,
	}, nil
}

func (c *WalletServer) Deposit(ctx context.Context, req *pb.DepositReq) (*pb.DepositRes, error) {
	userID, err := uuid.FromBytes(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id format")
	}

	tx, err := c.useCase.Deposit(ctx, userID, req.Amount, req.Description)
	if err != nil {
		return nil, err
	}

	wallet, _ := c.useCase.GetBalance(ctx, userID)
	var newBalance int64
	if wallet != nil {
		newBalance = wallet.Balance
	}

	return &pb.DepositRes{
		TransactionId: tx.ID[:],
		NewBalance:    newBalance,
	}, nil
}

func (c *WalletServer) Transfer(ctx context.Context, req *pb.TransferReq) (*pb.TransferRes, error) {
	fromUser, err := uuid.FromBytes(req.FromUserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid from_user_id format")
	}

	toUser, err := uuid.FromBytes(req.ToUserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid to_user_id format")
	}

	refID := req.ReferenceId
	if refID == "" {
		refID = uuid.New().String()
	}

	err = c.useCase.TransferMoney(ctx, fromUser, toUser, req.Amount, refID)
	if err != nil {
		return nil, err
	}

	return &pb.TransferRes{
		TransactionId: []byte(refID),
		Success:       true,
	}, nil
}

func (c *WalletServer) SearchWallets(ctx context.Context, req *pb.SearchWalletsReq) (*pb.SearchWalletsRes, error) {
	if c.esEngine == nil {
		return nil, status.Errorf(codes.Unimplemented, "elasticsearch engine is not available")
	}

	params := &search.SearchWalletParams{
		Query:    req.Query,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	}
	if req.UserType != nil {
		t := uint8(*req.UserType)
		params.UserType = &t
	}
	if req.Status != nil {
		s := uint8(*req.Status)
		params.Status = &s
	}

	result, err := c.esEngine.SearchWallets(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search failed: %v", err)
	}

	res := &pb.SearchWalletsRes{
		Total:    result.Total,
		Page:     int32(result.Page),
		PageSize: int32(result.PageSize),
		Wallets:  make([]*pb.WalletInfo, 0, len(result.Hits)),
	}

	for _, doc := range result.Hits {
		res.Wallets = append(res.Wallets, c.mapper.ESWalletToProto(&doc))
	}

	return res, nil
}

func (c *WalletServer) SearchTransactions(ctx context.Context, req *pb.SearchTransactionsReq) (*pb.SearchTransactionsRes, error) {
	if c.esEngine == nil {
		return nil, status.Errorf(codes.Unimplemented, "elasticsearch engine is not available")
	}

	params := &search.SearchTransactionParams{
		Query:    req.Query,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	}

	if len(req.WalletId) > 0 {
		if wid, err := uuid.FromBytes(req.WalletId); err == nil {
			params.WalletID = &wid
		}
	}
	if req.TransactionType != nil {
		t := uint8(*req.TransactionType)
		params.TransactionType = &t
	}
	if req.Status != nil {
		s := uint8(*req.Status)
		params.Status = &s
	}

	result, err := c.esEngine.SearchTransactions(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search failed: %v", err)
	}

	res := &pb.SearchTransactionsRes{
		Total:        result.Total,
		Page:         int32(result.Page),
		PageSize:     int32(result.PageSize),
		Transactions: make([]*pb.TransactionInfo, 0, len(result.Hits)),
	}

	for _, doc := range result.Hits {
		res.Transactions = append(res.Transactions, c.mapper.ESTransactionToProto(&doc))
	}

	return res, nil
}
