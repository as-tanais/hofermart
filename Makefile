run-accrual:
	go run ./cmd/accrual

run-goph:
	go run ./cmd/gophermart -a=localhost:45422 -d postgres://db_user:pwd123@localhost:54323/hofermart?sslmode=disable -r=localhost:45423
