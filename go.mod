module inchworks.com/quiz

go 1.16

require (
	github.com/BurntSushi/toml v1.3.2 // indirect
	github.com/go-sql-driver/mysql v1.7.1
	github.com/golangcollege/sessions v1.2.0
	github.com/ilyakaznacheev/cleanenv v1.5.0
	github.com/inchworks/usage v1.3.0
	github.com/inchworks/webparts v1.4.1
	github.com/jmoiron/sqlx v1.3.5
	github.com/julienschmidt/httprouter v1.3.0
	github.com/justinas/alice v1.2.0
	github.com/justinas/nosurf v1.1.1
	github.com/microcosm-cc/bluemonday v1.0.25
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/image v0.13.0 // indirect
)

// replace github.com/inchworks/usage v1.3.0 => ../usage
// replace github.com/inchworks/webparts v1.4.1 => ../webparts
