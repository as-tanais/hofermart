run-a:
	go run ./cmd/accrual -a=:45423 -d postgres://db_user:pwd123@localhost:54323/hofermart?sslmode=disable

run-g:
	go run ./cmd/gophermart -a=:45422 -d postgres://db_user:pwd123@localhost:54323/hofermart?sslmode=disable -r=localhost:45423
