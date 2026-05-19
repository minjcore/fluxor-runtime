package telegram

import "context"

// Client is the main Telegram client interface
type Client interface {
	Messages() MessagesClient
	Chats() ChatsClient
	Users() UsersClient
}

// MessagesClient provides operations for Telegram messages
type MessagesClient interface {
	// Send sends a message
	Send(ctx context.Context, chatID int64, message *SendMessageRequest) (*Message, error)

	// SendPhoto sends a photo
	SendPhoto(ctx context.Context, chatID int64, photo *SendPhotoRequest) (*Message, error)

	// SendDocument sends a document
	SendDocument(ctx context.Context, chatID int64, doc *SendDocumentRequest) (*Message, error)

	// Edit edits a message
	Edit(ctx context.Context, chatID int64, messageID int, text string) (*Message, error)

	// Delete deletes a message
	Delete(ctx context.Context, chatID int64, messageID int) error

	// Forward forwards a message
	Forward(ctx context.Context, chatID, fromChatID int64, messageID int) (*Message, error)
}

// ChatsClient provides operations for Telegram chats
type ChatsClient interface {
	// Get gets chat info
	Get(ctx context.Context, chatID int64) (*Chat, error)

	// GetAdministrators gets chat administrators
	GetAdministrators(ctx context.Context, chatID int64) ([]ChatMember, error)

	// GetMemberCount gets chat member count
	GetMemberCount(ctx context.Context, chatID int64) (int, error)

	// GetMember gets a chat member
	GetMember(ctx context.Context, chatID, userID int64) (*ChatMember, error)

	// Leave leaves a chat
	Leave(ctx context.Context, chatID int64) error
}

// UsersClient provides operations for Telegram users
type UsersClient interface {
	// GetMe gets the bot user
	GetMe(ctx context.Context) (*User, error)

	// GetUserProfilePhotos gets user profile photos
	GetUserProfilePhotos(ctx context.Context, userID int64, offset, limit int) (*UserProfilePhotos, error)
}

// Message represents a Telegram message
type Message struct {
	MessageID       int              `json:"message_id"`
	From            *User            `json:"from,omitempty"`
	SenderChat      *Chat            `json:"sender_chat,omitempty"`
	Date            int64            `json:"date"`
	Chat            *Chat            `json:"chat"`
	ForwardFrom     *User            `json:"forward_from,omitempty"`
	ForwardFromChat *Chat            `json:"forward_from_chat,omitempty"`
	ForwardDate     int64            `json:"forward_date,omitempty"`
	ReplyToMessage  *Message         `json:"reply_to_message,omitempty"`
	EditDate        int64            `json:"edit_date,omitempty"`
	Text            string           `json:"text,omitempty"`
	Entities        []MessageEntity  `json:"entities,omitempty"`
	Photo           []PhotoSize      `json:"photo,omitempty"`
	Document        *Document        `json:"document,omitempty"`
	Video           *Video           `json:"video,omitempty"`
	Audio           *Audio           `json:"audio,omitempty"`
	Voice           *Voice           `json:"voice,omitempty"`
	Caption         string           `json:"caption,omitempty"`
	CaptionEntities []MessageEntity  `json:"caption_entities,omitempty"`
	Contact         *Contact         `json:"contact,omitempty"`
	Location        *Location        `json:"location,omitempty"`
	NewChatMembers  []User           `json:"new_chat_members,omitempty"`
	LeftChatMember  *User            `json:"left_chat_member,omitempty"`
	NewChatTitle    string           `json:"new_chat_title,omitempty"`
	NewChatPhoto    []PhotoSize      `json:"new_chat_photo,omitempty"`
	PinnedMessage   *Message         `json:"pinned_message,omitempty"`
	ReplyMarkup     *InlineKeyboard  `json:"reply_markup,omitempty"`
}

// SendMessageRequest represents parameters for sending a message
type SendMessageRequest struct {
	Text                  string          `json:"text"`
	ParseMode             string          `json:"parse_mode,omitempty"` // HTML, Markdown, MarkdownV2
	Entities              []MessageEntity `json:"entities,omitempty"`
	DisableWebPagePreview bool            `json:"disable_web_page_preview,omitempty"`
	DisableNotification   bool            `json:"disable_notification,omitempty"`
	ReplyToMessageID      int             `json:"reply_to_message_id,omitempty"`
	ReplyMarkup           interface{}     `json:"reply_markup,omitempty"`
}

// SendPhotoRequest represents parameters for sending a photo
type SendPhotoRequest struct {
	Photo               string          `json:"photo"` // file_id, URL, or path
	Caption             string          `json:"caption,omitempty"`
	ParseMode           string          `json:"parse_mode,omitempty"`
	CaptionEntities     []MessageEntity `json:"caption_entities,omitempty"`
	DisableNotification bool            `json:"disable_notification,omitempty"`
	ReplyToMessageID    int             `json:"reply_to_message_id,omitempty"`
	ReplyMarkup         interface{}     `json:"reply_markup,omitempty"`
}

// SendDocumentRequest represents parameters for sending a document
type SendDocumentRequest struct {
	Document            string          `json:"document"` // file_id, URL, or path
	Caption             string          `json:"caption,omitempty"`
	ParseMode           string          `json:"parse_mode,omitempty"`
	CaptionEntities     []MessageEntity `json:"caption_entities,omitempty"`
	DisableNotification bool            `json:"disable_notification,omitempty"`
	ReplyToMessageID    int             `json:"reply_to_message_id,omitempty"`
	ReplyMarkup         interface{}     `json:"reply_markup,omitempty"`
}

// MessageEntity represents a special entity in a message
type MessageEntity struct {
	Type     string `json:"type"` // mention, hashtag, url, bold, italic, code, pre, etc.
	Offset   int    `json:"offset"`
	Length   int    `json:"length"`
	URL      string `json:"url,omitempty"`
	User     *User  `json:"user,omitempty"`
	Language string `json:"language,omitempty"`
}

// User represents a Telegram user
type User struct {
	ID                      int64  `json:"id"`
	IsBot                   bool   `json:"is_bot"`
	FirstName               string `json:"first_name"`
	LastName                string `json:"last_name,omitempty"`
	Username                string `json:"username,omitempty"`
	LanguageCode            string `json:"language_code,omitempty"`
	IsPremium               bool   `json:"is_premium,omitempty"`
	AddedToAttachmentMenu   bool   `json:"added_to_attachment_menu,omitempty"`
	CanJoinGroups           bool   `json:"can_join_groups,omitempty"`
	CanReadAllGroupMessages bool   `json:"can_read_all_group_messages,omitempty"`
	SupportsInlineQueries   bool   `json:"supports_inline_queries,omitempty"`
}

// Chat represents a Telegram chat
type Chat struct {
	ID                    int64          `json:"id"`
	Type                  string         `json:"type"` // private, group, supergroup, channel
	Title                 string         `json:"title,omitempty"`
	Username              string         `json:"username,omitempty"`
	FirstName             string         `json:"first_name,omitempty"`
	LastName              string         `json:"last_name,omitempty"`
	Photo                 *ChatPhoto     `json:"photo,omitempty"`
	Bio                   string         `json:"bio,omitempty"`
	HasPrivateForwards    bool           `json:"has_private_forwards,omitempty"`
	Description           string         `json:"description,omitempty"`
	InviteLink            string         `json:"invite_link,omitempty"`
	PinnedMessage         *Message       `json:"pinned_message,omitempty"`
	Permissions           *ChatPermissions `json:"permissions,omitempty"`
	SlowModeDelay         int            `json:"slow_mode_delay,omitempty"`
	MessageAutoDeleteTime int            `json:"message_auto_delete_time,omitempty"`
	LinkedChatID          int64          `json:"linked_chat_id,omitempty"`
	Location              *ChatLocation  `json:"location,omitempty"`
}

// ChatPhoto represents a chat photo
type ChatPhoto struct {
	SmallFileID       string `json:"small_file_id"`
	SmallFileUniqueID string `json:"small_file_unique_id"`
	BigFileID         string `json:"big_file_id"`
	BigFileUniqueID   string `json:"big_file_unique_id"`
}

// ChatPermissions represents chat permissions
type ChatPermissions struct {
	CanSendMessages       bool `json:"can_send_messages,omitempty"`
	CanSendMediaMessages  bool `json:"can_send_media_messages,omitempty"`
	CanSendPolls          bool `json:"can_send_polls,omitempty"`
	CanSendOtherMessages  bool `json:"can_send_other_messages,omitempty"`
	CanAddWebPagePreviews bool `json:"can_add_web_page_previews,omitempty"`
	CanChangeInfo         bool `json:"can_change_info,omitempty"`
	CanInviteUsers        bool `json:"can_invite_users,omitempty"`
	CanPinMessages        bool `json:"can_pin_messages,omitempty"`
}

// ChatLocation represents a location to which a chat is connected
type ChatLocation struct {
	Location Location `json:"location"`
	Address  string   `json:"address"`
}

// ChatMember represents a chat member
type ChatMember struct {
	User                  *User  `json:"user"`
	Status                string `json:"status"` // creator, administrator, member, restricted, left, kicked
	CustomTitle           string `json:"custom_title,omitempty"`
	IsAnonymous           bool   `json:"is_anonymous,omitempty"`
	CanBeEdited           bool   `json:"can_be_edited,omitempty"`
	CanManageChat         bool   `json:"can_manage_chat,omitempty"`
	CanPostMessages       bool   `json:"can_post_messages,omitempty"`
	CanEditMessages       bool   `json:"can_edit_messages,omitempty"`
	CanDeleteMessages     bool   `json:"can_delete_messages,omitempty"`
	CanManageVideoChats   bool   `json:"can_manage_video_chats,omitempty"`
	CanRestrictMembers    bool   `json:"can_restrict_members,omitempty"`
	CanPromoteMembers     bool   `json:"can_promote_members,omitempty"`
	CanChangeInfo         bool   `json:"can_change_info,omitempty"`
	CanInviteUsers        bool   `json:"can_invite_users,omitempty"`
	CanPinMessages        bool   `json:"can_pin_messages,omitempty"`
	IsMember              bool   `json:"is_member,omitempty"`
	CanSendMessages       bool   `json:"can_send_messages,omitempty"`
	CanSendMediaMessages  bool   `json:"can_send_media_messages,omitempty"`
	CanSendPolls          bool   `json:"can_send_polls,omitempty"`
	CanSendOtherMessages  bool   `json:"can_send_other_messages,omitempty"`
	CanAddWebPagePreviews bool   `json:"can_add_web_page_previews,omitempty"`
	UntilDate             int64  `json:"until_date,omitempty"`
}

// PhotoSize represents a photo or file/sticker thumbnail
type PhotoSize struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int    `json:"file_size,omitempty"`
}

// Document represents a general file
type Document struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Thumb        *PhotoSize `json:"thumb,omitempty"`
	FileName     string     `json:"file_name,omitempty"`
	MimeType     string     `json:"mime_type,omitempty"`
	FileSize     int        `json:"file_size,omitempty"`
}

// Video represents a video file
type Video struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Width        int        `json:"width"`
	Height       int        `json:"height"`
	Duration     int        `json:"duration"`
	Thumb        *PhotoSize `json:"thumb,omitempty"`
	FileName     string     `json:"file_name,omitempty"`
	MimeType     string     `json:"mime_type,omitempty"`
	FileSize     int        `json:"file_size,omitempty"`
}

// Audio represents an audio file
type Audio struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Duration     int        `json:"duration"`
	Performer    string     `json:"performer,omitempty"`
	Title        string     `json:"title,omitempty"`
	FileName     string     `json:"file_name,omitempty"`
	MimeType     string     `json:"mime_type,omitempty"`
	FileSize     int        `json:"file_size,omitempty"`
	Thumb        *PhotoSize `json:"thumb,omitempty"`
}

// Voice represents a voice note
type Voice struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Duration     int    `json:"duration"`
	MimeType     string `json:"mime_type,omitempty"`
	FileSize     int    `json:"file_size,omitempty"`
}

// Contact represents a phone contact
type Contact struct {
	PhoneNumber string `json:"phone_number"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name,omitempty"`
	UserID      int64  `json:"user_id,omitempty"`
	VCard       string `json:"vcard,omitempty"`
}

// Location represents a point on the map
type Location struct {
	Longitude            float64 `json:"longitude"`
	Latitude             float64 `json:"latitude"`
	HorizontalAccuracy   float64 `json:"horizontal_accuracy,omitempty"`
	LivePeriod           int     `json:"live_period,omitempty"`
	Heading              int     `json:"heading,omitempty"`
	ProximityAlertRadius int     `json:"proximity_alert_radius,omitempty"`
}

// InlineKeyboard represents an inline keyboard
type InlineKeyboard struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// InlineKeyboardButton represents a button in an inline keyboard
type InlineKeyboardButton struct {
	Text                         string `json:"text"`
	URL                          string `json:"url,omitempty"`
	CallbackData                 string `json:"callback_data,omitempty"`
	SwitchInlineQuery            string `json:"switch_inline_query,omitempty"`
	SwitchInlineQueryCurrentChat string `json:"switch_inline_query_current_chat,omitempty"`
}

// UserProfilePhotos represents a user's profile photos
type UserProfilePhotos struct {
	TotalCount int           `json:"total_count"`
	Photos     [][]PhotoSize `json:"photos"`
}

// APIResponse represents the Telegram API response
type APIResponse struct {
	OK          bool        `json:"ok"`
	Result      interface{} `json:"result,omitempty"`
	Description string      `json:"description,omitempty"`
	ErrorCode   int         `json:"error_code,omitempty"`
}
