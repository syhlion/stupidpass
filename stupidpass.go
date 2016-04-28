package main

import (
	"database/sql"
	"flag"
	"log"
	"math/rand"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tucnak/telebot"
)

const PASS_LENGTH = 10

var (
	db                *sql.DB
	token             = flag.String("t", "", "input telegram bot token")
	letterRunes       = []rune("23456789")
	secondletterRunes = []rune("abcdefghijkmnpqrstwxy")
	firstletterRunes  = []rune("ABCDEFGHJKMNQRSTWXY")
)

func RandString(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)

	b[0] = firstletterRunes[rand.Intn(len(firstletterRunes))]
	b[1] = secondletterRunes[rand.Intn(len(secondletterRunes))]
	for i := 2; i < len(b); i++ {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func ResetCommand(m telebot.Message) (pass string, err error) {
generatePassword:
	pass = RandString(PASS_LENGTH)
	query := "SELECT pass FROM stupidpass WHERE uid = ? AND pass = ?"
	stmt, err := db.Prepare(query)
	if err != nil {
		log.Println(err)
		return
	}
	err = stmt.QueryRow(m.Sender.ID, pass).Scan(&pass)

	defer stmt.Close()
	if err == nil {
		goto generatePassword
	}

	cmd := "INSERT INTO stupidpass (uid,pass,date) VALUES (?,?,?)"
	tx, err := db.Begin()
	if err != nil {
		return
	}
	stmt, err = tx.Prepare(cmd)
	if err != nil {
		return
	}
	date := time.Now().Format("2006/01/02 15:04:05")

	_, err = stmt.Exec(m.Sender.ID, pass, date)
	if err != nil {
		tx.Rollback()
		stmt.Close()
		return
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		stmt.Close()
		return
	}

	cmd = "SELECT count(*) FROM stupidpass WHERE uid =?"
	stmt, err = db.Prepare(cmd)
	if err != nil {
		log.Println(err)
		return
	}
	var count int
	stmt.QueryRow(m.Sender.ID).Scan(&count)
	if count >= 5 {

		cmd = "SELECT pass FROM stupidpass WHERE uid =? ORDER BY DATE ASC limit 1"
		stmt, err = db.Prepare(cmd)
		if err != nil {
			log.Println(err)
			return
		}
		var dpass string
		err = stmt.QueryRow(m.Sender.ID).Scan(&dpass)
		if err != nil {
			log.Println("aabb", err)
			return
		}
		cmd = "DELETE FROM stupidpass WHERE uid = ? AND pass = ?"
		stmt, err = db.Prepare(cmd)
		if err != nil {
			log.Println(err)
			return
		}
		_, err = stmt.Exec(m.Sender.ID, dpass)

	}

	return
}

func PasswordCommand(m telebot.Message) (pass string, err error) {
	query := "SELECT pass FROM stupidpass WHERE uid = ? ORDER BY date DESC limit 1"
	stmt, err := db.Prepare(query)
	if err != nil {
		log.Println(err)
		return
	}
	err = stmt.QueryRow(m.Sender.ID).Scan(&pass)
	defer stmt.Close()
	if err == nil {
		return
	}
	pass = RandString(PASS_LENGTH)

	cmd := "INSERT INTO stupidpass (uid,pass,date) VALUES (?,?,?)"
	tx, err := db.Begin()
	if err != nil {
		return
	}
	stmt, err = tx.Prepare(cmd)
	if err != nil {
		return
	}
	date := time.Now().Format("2006/01/02 15:04:05")

	_, err = stmt.Exec(m.Sender.ID, pass, date)
	if err != nil {
		tx.Rollback()
		stmt.Close()
		return
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		stmt.Close()
		return
	}
	return
}

func HelpCommand(m telebot.Message) (msg string, err error) {
	msg = "\n 這是笨蛋密碼產生機器人，您可以照著指令設定 \n /password - 查詢現在密碼（如無密碼會產生一組）\n /reset - 重新設定密碼(會重新產生一組新密碼)"
	return
}

func init() {
	d, err := sql.Open("sqlite3", "./db.sqlite3")
	if err != nil {
		return
	}

	sqlStmt := `
	create table if not exists stupidpass (uid,pass,date,UNIQUE (uid,pass) ON CONFLICT REPLACE)
	`
	_, err = d.Exec(sqlStmt)
	if err != nil {
		panic(err)
	}
	db = d
}
func main() {
	flag.Parse()
	bot, err := telebot.NewBot(*token)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Bot Start")

	messages := make(chan telebot.Message)
	bot.Listen(messages, 60*time.Second)
	for message := range messages {
		log.Println(message.Sender.ID, message.Sender.Username, message.Text)
		switch message.Text {
		case "/help":
			s, _ := HelpCommand(message)
			bot.SendMessage(message.Chat, s, nil)
			break
		case "/password":
			s, _ := PasswordCommand(message)
			bot.SendMessage(message.Chat, s, nil)
			break
		case "/reset":
			s, _ := ResetCommand(message)
			bot.SendMessage(message.Chat, s, nil)
			break
		default:
			bot.SendMessage(message.Chat, "請輸入正確指令", nil)
			break
		}

	}
}
