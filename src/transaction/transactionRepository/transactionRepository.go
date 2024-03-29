package transactionRepository

import (
	"bike-rent-express/model/dto/transactionDto"
	"bike-rent-express/src/transaction"
	"database/sql"
	"errors"
	"time"
)

type transactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) transaction.TransactionRepository {
	return &transactionRepository{db}
}

func (t *transactionRepository) Add(transactionRequest transactionDto.AddTransactionRequest) (transactionDto.AddTransactionRequest, error) {
	tx, err := t.db.Begin()
	if err != nil {
		tx.Rollback()
		return transactionRequest, err
	}

	startDate, err := time.Parse("02-01-2006", transactionRequest.StartDate)
	if err != nil {
		tx.Rollback()
		return transactionRequest, err
	}

	endDate, err := time.Parse("02-01-2006", transactionRequest.EndDate)
	if err != nil {
		tx.Rollback()
		return transactionRequest, err
	}

	difference := endDate.Sub(startDate).Hours() / 24
	if difference < 1 {
		tx.Rollback()
		return transactionRequest, errors.New("end date is at least 1 day from the start date")
	}

	query := "SELECT price FROM motor_vehicle WHERE id = $1 AND status = 'AVAILABLE';"
	priceMotor := 0

	err = tx.QueryRow(query, transactionRequest.MotorVehicleId).Scan(&priceMotor)
	if err != nil {
		tx.Rollback()
		return transactionRequest, errors.New("1")
	}
	priceMotor *= int(difference)

	query = "SELECT amount FROM balance WHERE user_id = $1;"
	userBalance := 0

	err = tx.QueryRow(query, transactionRequest.UserID).Scan(&userBalance)
	if err != nil {
		tx.Rollback()
		return transactionRequest, err
	}

	if userBalance < priceMotor {
		tx.Rollback()
		return transactionRequest, errors.New("2")
	}

	result := userBalance - priceMotor

	query = "UPDATE balance SET amount = $1 WHERE user_id = $2;"
	_, err = tx.Exec(query, result, transactionRequest.UserID)
	if err != nil {
		tx.Rollback()
		return transactionRequest, err
	}

	query = "UPDATE motor_vehicle SET status = 'NOT_AVAILABLE' WHERE id = $1"
	_, err = tx.Exec(query, transactionRequest.MotorVehicleId)
	if err != nil {
		tx.Rollback()
		return transactionRequest, err
	}

	query = "UPDATE users SET can_rent = false WHERE id = $1"
	_, err = tx.Exec(query, transactionRequest.UserID)
	if err != nil {
		tx.Rollback()
		return transactionRequest, err
	}

	query = "INSERT INTO transaction(user_id, motor_vehicle_id, employee_id, start_date, end_date, price ) VALUES($1, $2, $3, $4, $5, $6) RETURNING id;"

	err = tx.QueryRow(query, transactionRequest.UserID, transactionRequest.MotorVehicleId, transactionRequest.EmployeeId, startDate, endDate, priceMotor).Scan(&transactionRequest.ID)
	if err != nil {
		tx.Rollback()
		return transactionRequest, err
	}
	tx.Commit()

	return transactionRequest, nil
}

func (t *transactionRepository) GetById(id string) (transactionDto.Transaction, error) {
	var transaction transactionDto.Transaction
	query := "SELECT id, user_id, motor_vehicle_id, start_date, end_date, price, created_at, updated_at, employee_id FROM transaction WHERE id = $1;"

	if err := t.db.QueryRow(query, id).Scan(&transaction.ID, &transaction.UserID, &transaction.MotorVehicleId, &transaction.StartDate, &transaction.EndDate, &transaction.Price, &transaction.CreatedAt, &transaction.UpdatedAt, &transaction.EmployeeId); err != nil {
		return transaction, err
	}
	return transaction, nil
}

func (t *transactionRepository) GetAll() ([]transactionDto.Transaction, error) {
	var transactions []transactionDto.Transaction

	query := "SELECT id, user_id, motor_vehicle_id, start_date, end_date, price, created_at, updated_at, employee_id FROM transaction;"

	row, err := t.db.Query(query)
	if err != nil {
		return transactions, err
	}
	defer row.Close()

	for row.Next() {
		var transaction transactionDto.Transaction
		if err := row.Scan(&transaction.ID, &transaction.UserID, &transaction.MotorVehicleId, &transaction.StartDate, &transaction.EndDate, &transaction.Price, &transaction.CreatedAt, &transaction.UpdatedAt, &transaction.EmployeeId); err != nil {
			return transactions, err
		}
		transactions = append(transactions, transaction)
	}

	return transactions, nil
}
