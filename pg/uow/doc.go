// Package uow provides a generic Unit of Work (UoW) and database execution
// engine for PostgreSQL using pgx/v5.
//
// This package is designed for Go Modular Monoliths and Clean Architecture.
// It uses context.Context to transparently propagate database transactions
// across multiple repository calls without leaking SQL or infrastructure
// details into the application (business) layer.
//
// # Concept
//
// The UnitOfWork injects a pgx.Tx into the context. When a repository executes
// a query, it uses GetExecutor to check the context. If a transaction is found,
// the query runs inside the transaction. If not, it falls back to the standard
// connection pool.
//
// # Example: Application Layer Port
//
// In your domain or application layer, define the interface so your service
// does not depend on this package directly:
//
//	package ports
//
//	import "context"
//
//	type UnitOfWork interface {
//	    WithTransaction(ctx context.Context, fn func(txCtx context.Context) error) error
//	}
//
// # Example: Infrastructure Layer (Repository)
//
// Your repositories use GetExecutor to seamlessly support both standalone
// queries and transaction-bound queries:
//
//	package repository
//
//	import (
//	    "context"
//	    "github.com/jackc/pgx/v5/pgxpool"
//	    "github.com/ovya/ogl/postgresql/uow"
//	)
//
//	type UserRepository struct {
//	    pool *pgxpool.Pool
//	}
//
//	func (r *UserRepository) Save(ctx context.Context, user *User) error {
//	    // Magically uses the transaction if it exists in the context!
//	    exec := uow.GetExecutor(ctx, r.pool)
//
//	    _, err := exec.Exec(ctx, "INSERT INTO users (id, name) VALUES ($1, $2)", user.ID, user.Name)
//	    return err
//	}
//
// # Example: Application Service (Use Case)
//
// The application service coordinates the transaction without knowing about SQL:
//
//	func (s *UserService) RegisterUser(ctx context.Context, req Request) error {
//	    user := domain.NewUser(req.Name)
//
//	    // Both Save and Dispatch will execute within the same PostgreSQL transaction.
//	    // If either fails, the entire block is rolled back automatically.
//	    return s.uow.WithTransaction(ctx, func(txCtx context.Context) error {
//	        if err := s.userRepo.Save(txCtx, user); err != nil {
//	            return err
//	        }
//
//	        if err := s.eventDispatcher.Dispatch(txCtx, user.Events()); err != nil {
//	            return err
//	        }
//
//	        return nil
package uow
