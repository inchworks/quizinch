module inchworks.com/quiz

go 1.16

require (
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golangcollege/sessions v1.2.0
	github.com/ilyakaznacheev/cleanenv v1.2.6
	github.com/inchworks/usage v1.2.0
	github.com/inchworks/webparts v1.0.0
	github.com/jmoiron/sqlx v1.3.4
	github.com/julienschmidt/httprouter v1.3.0
	github.com/justinas/alice v1.2.0
	github.com/justinas/nosurf v1.1.1
	github.com/microcosm-cc/bluemonday v1.0.16
)

// replace github.com/inchworks/usage v1.2.0 => ../../usage
// replace github.com/inchworks/webparts v1.0.0 => ../../webparts
