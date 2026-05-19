package slack

import "context"

// Client is the main Slack client interface
type Client interface {
	Messages() MessagesClient
	Channels() ChannelsClient
	Users() UsersClient
}

// MessagesClient provides operations for Slack messages
type MessagesClient interface {
	// Send sends a message to a channel
	Send(ctx context.Context, channelID string, message *Message) (*Message, error)

	// Update updates an existing message
	Update(ctx context.Context, channelID, timestamp string, message *Message) (*Message, error)

	// Delete deletes a message
	Delete(ctx context.Context, channelID, timestamp string) error

	// GetHistory retrieves message history for a channel
	GetHistory(ctx context.Context, channelID string, params HistoryParams) ([]Message, error)
}

// ChannelsClient provides operations for Slack channels
type ChannelsClient interface {
	// List returns all channels
	List(ctx context.Context, params ListChannelsParams) ([]Channel, error)

	// Get returns a specific channel
	Get(ctx context.Context, channelID string) (*Channel, error)

	// Create creates a new channel
	Create(ctx context.Context, name string, isPrivate bool) (*Channel, error)

	// Archive archives a channel
	Archive(ctx context.Context, channelID string) error

	// Invite invites a user to a channel
	Invite(ctx context.Context, channelID, userID string) error
}

// UsersClient provides operations for Slack users
type UsersClient interface {
	// List returns all users
	List(ctx context.Context) ([]User, error)

	// Get returns a specific user
	Get(ctx context.Context, userID string) (*User, error)

	// GetByEmail returns a user by email
	GetByEmail(ctx context.Context, email string) (*User, error)
}

// Message represents a Slack message
type Message struct {
	// Channel ID
	Channel string `json:"channel,omitempty"`

	// Message text
	Text string `json:"text"`

	// Timestamp (message ID)
	Timestamp string `json:"ts,omitempty"`

	// User ID of the sender
	User string `json:"user,omitempty"`

	// Bot ID if sent by a bot
	BotID string `json:"bot_id,omitempty"`

	// Thread timestamp (for replies)
	ThreadTS string `json:"thread_ts,omitempty"`

	// Attachments
	Attachments []Attachment `json:"attachments,omitempty"`

	// Blocks (Block Kit elements)
	Blocks []Block `json:"blocks,omitempty"`

	// Message subtype
	Subtype string `json:"subtype,omitempty"`
}

// Attachment represents a Slack message attachment
type Attachment struct {
	Color      string   `json:"color,omitempty"`
	Fallback   string   `json:"fallback,omitempty"`
	Title      string   `json:"title,omitempty"`
	TitleLink  string   `json:"title_link,omitempty"`
	Text       string   `json:"text,omitempty"`
	Pretext    string   `json:"pretext,omitempty"`
	AuthorName string   `json:"author_name,omitempty"`
	AuthorLink string   `json:"author_link,omitempty"`
	AuthorIcon string   `json:"author_icon,omitempty"`
	ImageURL   string   `json:"image_url,omitempty"`
	ThumbURL   string   `json:"thumb_url,omitempty"`
	Footer     string   `json:"footer,omitempty"`
	FooterIcon string   `json:"footer_icon,omitempty"`
	Timestamp  int64    `json:"ts,omitempty"`
	Fields     []Field  `json:"fields,omitempty"`
	Actions    []Action `json:"actions,omitempty"`
}

// Field represents an attachment field
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// Action represents an attachment action
type Action struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	URL   string `json:"url,omitempty"`
	Style string `json:"style,omitempty"`
}

// Block represents a Block Kit block
type Block struct {
	Type     string                 `json:"type"`
	BlockID  string                 `json:"block_id,omitempty"`
	Text     *TextObject            `json:"text,omitempty"`
	Elements []interface{}          `json:"elements,omitempty"`
	Fields   []TextObject           `json:"fields,omitempty"`
	Accessory map[string]interface{} `json:"accessory,omitempty"`
}

// TextObject represents a text object in Block Kit
type TextObject struct {
	Type     string `json:"type"` // "plain_text" or "mrkdwn"
	Text     string `json:"text"`
	Emoji    bool   `json:"emoji,omitempty"`
	Verbatim bool   `json:"verbatim,omitempty"`
}

// Channel represents a Slack channel
type Channel struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	IsChannel      bool     `json:"is_channel"`
	IsGroup        bool     `json:"is_group"`
	IsIM           bool     `json:"is_im"`
	IsPrivate      bool     `json:"is_private"`
	IsMPIM         bool     `json:"is_mpim"`
	IsArchived     bool     `json:"is_archived"`
	IsGeneral      bool     `json:"is_general"`
	IsMember       bool     `json:"is_member"`
	Topic          Topic    `json:"topic"`
	Purpose        Purpose  `json:"purpose"`
	NumMembers     int      `json:"num_members"`
	MemberIDs      []string `json:"members,omitempty"`
	Created        int64    `json:"created"`
	Creator        string   `json:"creator"`
}

// Topic represents a channel topic
type Topic struct {
	Value   string `json:"value"`
	Creator string `json:"creator"`
	LastSet int64  `json:"last_set"`
}

// Purpose represents a channel purpose
type Purpose struct {
	Value   string `json:"value"`
	Creator string `json:"creator"`
	LastSet int64  `json:"last_set"`
}

// User represents a Slack user
type User struct {
	ID                string  `json:"id"`
	TeamID            string  `json:"team_id"`
	Name              string  `json:"name"`
	RealName          string  `json:"real_name"`
	DisplayName       string  `json:"display_name,omitempty"`
	Email             string  `json:"email,omitempty"`
	IsAdmin           bool    `json:"is_admin"`
	IsOwner           bool    `json:"is_owner"`
	IsBot             bool    `json:"is_bot"`
	IsAppUser         bool    `json:"is_app_user"`
	IsRestricted      bool    `json:"is_restricted"`
	IsUltraRestricted bool    `json:"is_ultra_restricted"`
	Deleted           bool    `json:"deleted"`
	Profile           Profile `json:"profile"`
	Timezone          string  `json:"tz"`
	TimezoneLabel     string  `json:"tz_label"`
	TimezoneOffset    int     `json:"tz_offset"`
}

// Profile represents a user's profile
type Profile struct {
	Title                 string `json:"title"`
	Phone                 string `json:"phone"`
	Skype                 string `json:"skype"`
	RealName              string `json:"real_name"`
	RealNameNormalized    string `json:"real_name_normalized"`
	DisplayName           string `json:"display_name"`
	DisplayNameNormalized string `json:"display_name_normalized"`
	StatusText            string `json:"status_text"`
	StatusEmoji           string `json:"status_emoji"`
	Email                 string `json:"email"`
	FirstName             string `json:"first_name"`
	LastName              string `json:"last_name"`
	Image24               string `json:"image_24"`
	Image32               string `json:"image_32"`
	Image48               string `json:"image_48"`
	Image72               string `json:"image_72"`
	Image192              string `json:"image_192"`
	Image512              string `json:"image_512"`
}

// HistoryParams contains parameters for message history
type HistoryParams struct {
	Limit     int    `json:"limit,omitempty"`
	Cursor    string `json:"cursor,omitempty"`
	Latest    string `json:"latest,omitempty"`
	Oldest    string `json:"oldest,omitempty"`
	Inclusive bool   `json:"inclusive,omitempty"`
}

// ListChannelsParams contains parameters for listing channels
type ListChannelsParams struct {
	Limit           int    `json:"limit,omitempty"`
	Cursor          string `json:"cursor,omitempty"`
	ExcludeArchived bool   `json:"exclude_archived,omitempty"`
	Types           string `json:"types,omitempty"` // "public_channel,private_channel"
}

// APIResponse is the base response from Slack API
type APIResponse struct {
	OK       bool   `json:"ok"`
	Error    string `json:"error,omitempty"`
	Warning  string `json:"warning,omitempty"`
	Metadata struct {
		NextCursor string `json:"next_cursor,omitempty"`
	} `json:"response_metadata,omitempty"`
}
