package tgclient

type UpdatesResponse struct {
	CommonResponse
	Updates []Update `json:"result"`
}

type WithMessageResponse struct {
	CommonResponse
	Message Message `json:"result"`
}

type WithAdminsResponse struct {
	CommonResponse
	Admins []Admin `json:"result"`
}

type Admin struct {
	User User `json:"user"`
}

type CommonResponse struct {
	Ok        bool   `json:"ok"`
	ErrorCode int    `json:"error_code,omitempty"`
	Descr     string `json:"description,omitempty"`
}

type Update struct {
	Id       int           `json:"update_id"`
	Message  Message       `json:"message"`
	Callback CallbackQuery `json:"callback_query,omitempty"`
}

type Message struct {
	Id       int64    `json:"message_id"`
	From     User     `json:"from"`
	Chat     Chat     `json:"chat"`
	Date     int64    `json:"date"`
	ThreadId int64    `json:"message_thread_id"`
	Text     string   `json:"text"`
	Entities []Enitiy `json:"entities"`

	Keyboard *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

const (
	ChatTypePrivate    = "private"
	ChatTypeSupergroup = "supergroup"
)

type Chat struct {
	Id   int64  `json:"id"`
	Type string `json:"type"`
}

const EntityBotCommand = "bot_command"

type Enitiy struct {
	Offset int    `json:"offset"`
	Len    int    `json:"length"`
	Type   string `json:"type"`
}

type User struct {
	Id       int64  `json:"id"`
	Name     string `json:"first_name"`
	Username string `json:"username"`
}

type CallbackQuery struct {
	From    User    `json:"from"`
	Data    string  `json:"data"`
	Message Message `json:"message"`
}

type SendMessageParams struct {
	ChatId    int64  `json:"chat_id"`
	ThreadId  int64  `json:"message_thread_id,omitempty"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`

	Keyboard *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

type EditMessageParams struct {
	SendMessageParams
	MessageId int64 `json:"message_id"`
}

type SetCommandsParams struct {
	Commands []Command    `json:"commands"`
	Scope    CommandScope `json:"scope"`
}

type Command struct {
	Cmd   string `json:"command"`
	Descr string `json:"description"`
}

type CommandScope struct {
	Type   string `json:"type"`
	ChatId int64  `json:"chat_id,omitempty"`
}

type InlineKeyboardMarkup struct {
	Keyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type InlineKeyboardButton struct {
	Text string `json:"text"`
	Data string `json:"callback_data"`
}
