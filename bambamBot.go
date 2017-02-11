/*
    Uses go-sqlte3 -> https://github.com/mattn/go-sqlite3
    Uses telegram-bot-api -> https://gopkg.in/telegram-bot-api.v4

    Based in bot.go -> https://github.com/ReiGelado/GoLangCodingBot

    30 dec 2016 -> Sam Uel
*/

package main

import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    "gopkg.in/telegram-bot-api.v4"
    "io/ioutil"
    "encoding/json"
    "strings"
    "log"
)

const (
    BOT_VERSION = "v0.3"
    MEMBER_TABLE = "members"
    BANNED_TABLE = "banned_members"
)

var (
    config []groupConfig
    userID int
)

//groupConfig will be used to create a structure that will receive unmarshal json
type groupConfig struct {
    Token string `json:"token,omitempty"` //Token
    AdminID []int `json:"adminID, omitempty"` //Array of id's
}

//startBot initializes bot through the Token received and returns a BotAPI instance
func startBot(token string) (bot *tgbotapi.BotAPI) {
    bot, err := tgbotapi.NewBotAPI(token)
    errorCheck(err)
    bot.Debug = false
    return bot
}

//errorCheck is to verify if there is an error
func errorCheck(err error) {
    if err != nil {
        log.Fatal("[ERROR]->", err)
    }
}

//readFile reads the file named by filename and returns the contents
func readFile(filename string) []byte {
    file, _ := ioutil.ReadFile(filename)
    return file
}

//----------Database------------
func initDb() (db *sql.DB) {
    db, err := sql.Open("sqlite3", "./bambamdb/db")
    errorCheck(err)
    return db
}

//insertMember inserts a member into the database
func insertMember(db sql.DB, userID int, username, table string) {
    tx, err := db.Begin()
    errorCheck(err)
    stmt, err := tx.Prepare("INSERT INTO "+ table +"(username,user_id) VALUES(?,?)")
    errorCheck(err)
    stmt.Exec(username, userID)
    tx.Commit()
}

//selectMember selects a member from the database and returns user id
func selectMember(db sql.DB, username, table string) (userID int) {
    rows, err := db.Query("SELECT user_id FROM "+ table +" WHERE username=?", username)
    errorCheck(err)
    for rows.Next() {
        err = rows.Scan(&userID)
        errorCheck(err)
    }
    err = rows.Err()
    errorCheck(err)
    defer rows.Close()
    return userID
}
//------------------------------

/*stringPrepare receives two strings(str, command) and returns str formatted
  trim str prefix(command), trim space and replace @(If there is '@') by "" */
func stringPrepare(str, command string) string {
    str = strings.TrimPrefix(str, command)
    str = strings.TrimSpace(str)

    if strings.Contains(str, "@") {
        str = strings.Replace(str, "@", "", -1)
    }
    return str
}

//stringCompare receives two strings(message, str), compare them and returns a boolean value 
func stringCompare(message, str string) bool {
    if strings.Contains(strings.ToUpper(string(message)), str) {
        return true
    }
    return false
}

//isCommand is to verify if is a bot command and returns a boolean value
func isCommand(message string) bool {
    if string(message[0]) == "/" {
        return true
    }
    return false
}

//adminPrivilege is to verify if there is an user id equal to admin id in the config 
func adminPrivilege(userID int) bool {
    for _, id := range config[0].AdminID {
        if userID == id {
            return true
        }
    }
    return false
}

//sendMessage only sends messages
func sendMessage(chatID int64, message string, bot *tgbotapi.BotAPI) {
    msg := tgbotapi.NewMessage(chatID, message)
    msg.ParseMode = "html"
    bot.Send(msg)
}

//kick removes a member from the chat
func kick(userID int, chatID int64, bot *tgbotapi.BotAPI) {
    kickConf := tgbotapi.ChatMemberConfig {
        ChatID: chatID,
        UserID: userID,
    }
    bot.KickChatMember(kickConf)
}

//kickMemberto calls kick function after verifying admin privilege
func kickMember(adminID int, chatID int64, message string, db sql.DB, bot *tgbotapi.BotAPI) string {
    username := stringPrepare(message, "/kick")
    userID := selectMember(db, username, MEMBER_TABLE)

    if adminPrivilege(adminID) {
        if userID != 0 {
            kick(userID, chatID, bot)
            return "<b>"+ username +"</b> Kickado!\nBIRLLLLLLLLLL!!!"
        }
        return "Usuário não encontrado na base de dados do <b>MUTANTE!</b>"
    }
    return "/kick é para os monstros, seu frango!\nVai pro Mural dos Frangos agora."
}

//banMember calls kick function after inserting a member into the banned members(database table) 
func banMember(adminID int, chatID int64, message string, db sql.DB, bot *tgbotapi.BotAPI) string {
    username := stringPrepare(message, "/ban")
    userID := selectMember(db, username, MEMBER_TABLE)

    if adminPrivilege(adminID) {
        if userID != 0 {
            insertMember(db, userID, username, BANNED_TABLE)
            kick(userID, chatID, bot)
            return "<b>"+ username +"</b>, quer subir em árvore porra?!\nTomou ban pra largar de ser frango.\nBIRLLLLLLLLLL!!!"
        }
        return "Usuário não encontrado na base de dados do <b>MUTANTE!</b>"
    }
    //--Mural dos Frangos-- Not implemented yet.
    return "Só monstros podem dar BAN, seu frango!\nVai pro Mural dos Frangos agora."
}

//getAdminsGroup returns admins
func getAdminsGroup(chatID int64, bot *tgbotapi.BotAPI) string {
    var adminsConcat string

    chatConf := tgbotapi.ChatConfig{
		ChatID: chatID,
    }

    admins, err := bot.GetChatAdministrators(chatConf)
    errorCheck(err)

    for i := range admins {
        adminsConcat += admins[i].User.UserName+", "
    }
    return "<b>Admins: </b>"+ adminsConcat
}

//---------botCommands----------
func botCommands(isPrivate bool, chatID int64, userID int, message, username string, db *sql.DB, bot *tgbotapi.BotAPI) string {
    if !isPrivate {
        if stringCompare(message, "/ADMINS") {
            return getAdminsGroup(chatID, bot)
        } else if stringCompare(message, "/BAN") {
            return banMember(userID, chatID, message, *db, bot)
        } else if stringCompare(message, "/KICK"){
            return kickMember(userID, chatID, message, *db, bot)
        } else if stringCompare(message, "/REGRAS") {
            return string(readFile("rules.txt"))
        } else if stringCompare(message, "/HELP") {
            return string(readFile("help.txt"))
        }
        return "Vai dá não <b>MUTANTE</b>, pois esse comando não existe!!!"
    } else {
        if stringCompare(message, "/HELP") || stringCompare(message, "/START") {
            return string(readFile("help.txt"))
        }
        return "Vai dá não <b>MUTANTE</b>, pois esse comando não existe!!!"
    }
}
//------------------------------

//-----horaDoshow messages------
func horaDoShow(chatID int64, userID int, message string, username string, bot *tgbotapi.BotAPI) string {
    if stringCompare(message, "NÃO VAI DAR") {
        return "<b>QUE NÃO VAI DAR!\nSAÍ DE CASA COMI PRA CARALHO PORRA!</b>"
    } else if stringCompare(message, "KKKK") {
        return "<b>Bora cumpade!\nSegura o maluco que tá doente! kkkkkkkkkkkk</b>"
    }
    return ""
}
//------------------------------

func main() {
    jsonConfig := readFile("config/config.json")

    err := json.Unmarshal(jsonConfig, &config)
    errorCheck(err)

    bot := startBot(config[0].Token)

    db := initDb()
    defer db.Close()

    log.Printf("Logged in [%s]", bot.Self.UserName)

    u := tgbotapi.NewUpdate(0)
    u.Timeout = 5

    updates, err := bot.GetUpdatesChan(u)
    errorCheck(err)
    for msg := range updates {
        if msg.Message == nil {
            continue
        }
        /*
            *** New Member Database Verification ***
            If the new member was banned from the group, Kick him.
            If not, will insert him into the database.
        */
        if msg.Message.NewChatMember != nil {
            newMember := msg.Message.NewChatMember.UserName
            if selectMember(*db, newMember, BANNED_TABLE) != 0 {
                kick(selectMember(*db, newMember, MEMBER_TABLE), msg.Message.Chat.ID, bot)
                log.Printf("[FRANGO](%s) tentou subir nas árvores do grupo, porém foi derrubado(a)!", newMember)
                msg.Message.NewChatMember = nil
            } else {
                if selectMember(*db, newMember, MEMBER_TABLE) == 0 {
                    sendMessage(msg.Message.Chat.ID, "<b>Fala FRANGO, seja bem vindo cumpade!</b>\n\n"+BOT_VERSION, bot)
                    insertMember(*db, msg.Message.NewChatMember.ID, newMember, MEMBER_TABLE)
                    log.Printf("[FRANGO](%s) ID:[%d] foi add ao grupo e cadastrado.", newMember, msg.Message.NewChatMember.ID)
                } else {
                    log.Printf("[FRANGO](%s) ID:[%d] retornou ao grupo.", newMember, msg.Message.NewChatMember.ID)
                }
            }
        }

        if msg.Message.LeftChatMember != nil {
            log.Printf("[%s] foi removido(a) do grupo.", msg.Message.LeftChatMember.UserName)
            sendMessage(msg.Message.Chat.ID, "<b>Vá com DEUS!!!</b>", bot)
        }

        if msg.Message.Text != "" {
            if isCommand(msg.Message.Text) == true {
                commandMsg := botCommands(msg.Message.Chat.IsPrivate(), msg.Message.Chat.ID, msg.Message.From.ID, msg.Message.Text, msg.Message.From.UserName, db, bot)
                sendMessage(msg.Message.Chat.ID, commandMsg, bot)
            } else {
                commandMsg := horaDoShow(msg.Message.Chat.ID, msg.Message.From.ID, msg.Message.Text, msg.Message.From.UserName, bot)
                if commandMsg != "" {
                    sendMessage(msg.Message.Chat.ID, commandMsg, bot)
                }
            }
        }

        log.Printf("[%s](%s)", msg.Message.From.UserName, msg.Message.Text)
    }
}
