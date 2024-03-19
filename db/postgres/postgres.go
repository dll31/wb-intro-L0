package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"wb-intro-l0/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	Ctx        context.Context
	Pool       *pgxpool.Pool
	ConnString string
	db         Database
}

type Database struct {
	Name     string
	Settings dbSettings
}

type dbSettings struct {
	username string
	password string
	host     string
	port     string
}

func (p *Postgres) CreateConnString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", p.db.Settings.username, p.db.Settings.password, p.db.Settings.host, p.db.Settings.port, p.db.Name)
}

func NewDb() *Database {
	username, _ := os.LookupEnv("POSTGRES_USER")
	password, _ := os.LookupEnv("PGPASSWORD")
	host, _ := os.LookupEnv("POSTGRES_HOST")
	port, _ := os.LookupEnv("POSTGRES_PORT")
	dbName, _ := os.LookupEnv("POSTGRES_DB")

	dbS := dbSettings{
		username: username,
		password: password,
		host:     host,
		port:     port,
	}

	db := Database{
		Name:     dbName,
		Settings: dbS,
	}

	return &db
}

func (p *Postgres) NewPool(db *Database) {

	p.db = *db
	p.Ctx = context.Background()
	p.ConnString = p.CreateConnString()

	var err error
	p.Pool, err = pgxpool.New(p.Ctx, p.ConnString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}
}

func (p *Postgres) ClosePool() {
	p.Pool.Close()
}

func (p *Postgres) Insert(data *model.Model) (err error) {
	insertString := fmt.Sprintf("INSERT INTO %s (order_uid, order_data) VALUES ", p.db.Name)
	jsonFields, err := json.Marshal(data.Fields)
	if err != nil {
		log.Println("Cannot marshal fields in insert")
		return
	}
	_, err = p.Pool.Exec(p.Ctx, (insertString + "($1, $2)"), data.Id, jsonFields)

	if err != nil {
		log.Printf("Cannot insert into db. Error: %s\n", err)
		return
	}
	return
}

func (p *Postgres) SelectLastNRows(n int) (data map[string]interface{}) {

	data = make(map[string]interface{}, n)

	rows, err := p.Pool.Query(context.Background(), "SELECT * FROM orders ORDER BY id DESC LIMIT $1", n)
	if err != nil {
		log.Fatalf("Error querying database: %v\n", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var orderUid string
		var orderData map[string]interface{}

		if err := rows.Scan(&id, &orderUid, &orderData); err != nil {
			log.Fatalf("Error scanning row: %v\n", err)
		}

		fmt.Printf("ID: %d, OrderUid: %s, OrderData: %v\n", id, orderUid, orderData)
		data[orderUid] = orderData
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating over rows: %v\n", err)
	}

	return data
}

// func selectFromDb(conn *pgxpool.Pool, m *model.Model) {
// 	selectString := "SELECT order_uid, order_data FROM "
// 	row := conn.QueryRow(context.Background(), selectString)
// 	var jsonFields []byte
// 	row.Scan(m.Id, jsonFields)
// 	err := m.Unmarshal(&jsonFields)
// 	if err != nil {
// 		log.Println("Cannot unmarshal in select from db")
// 	}
// }
