package transactionRepository

import (
	"bike-rent-express/model/dto/transactionDto"
	"bike-rent-express/src/transaction"
	"database/sql"
	"errors"
	"fmt"
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

	query := "SELECT price FROM motor_vechile WHERE id = $1 AND status = 'AVAILABLE';"
	priceMotor := 0

	err = tx.QueryRow(query, transactionRequest.MotorVehicleId).Scan(&priceMotor)
	if err != nil {
		tx.Rollback()
		return transactionRequest, errors.New("motor not available")
	}
	fmt.Println(difference)
	fmt.Println(priceMotor)
	priceMotor *= int(difference)
	fmt.Println(priceMotor)

	query = "SELECT amount FROM balance WHERE user_id = $1;"
	userBalance := 0

	err = tx.QueryRow(query, transactionRequest.UserID).Scan(&userBalance)
	if err != nil {
		tx.Rollback()
		return transactionRequest, err
	}

	if userBalance < priceMotor {
		tx.Rollback()
		return transactionRequest, errors.New("balance is not enought")
	}

	result := userBalance - priceMotor

	fmt.Println("a", result)
	query = "UPDATE balance SET amount = $1 WHERE user_id = $2;"
	_, err = tx.Exec(query, result, transactionRequest.UserID)
	if err != nil {
		tx.Rollback()
		return transactionRequest, err
	}

	query = "UPDATE motor_vechile SET status = 'NOT_AVAILABLE' WHERE id = $1"
	_, err = tx.Exec(query, transactionRequest.MotorVehicleId)
	if err != nil {
		tx.Rollback()
		return transactionRequest, err
	}

	query = "INSERT INTO transaction(user_id, motor_vechile_id, employee_id, start_date, end_date, price ) VALUES($1, $2, $3, $4, $5, $6) RETURNING id;"

	err = tx.QueryRow(query, transactionRequest.UserID, transactionRequest.MotorVehicleId, transactionRequest.EmployeeId, startDate, endDate, priceMotor).Scan(&transactionRequest.ID)
	if err != nil {
		tx.Rollback()
		return transactionRequest, err
	}
	tx.Commit()

	return transactionRequest, nil
}
