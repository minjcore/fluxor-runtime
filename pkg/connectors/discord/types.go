package discord

import "context"

// Client is the main Discord client interface
type Client interface {
	Messages() MessagesClient
	Channels() ChannelsClient
	Guilds() GuildsClient
	Users() UsersClient
}

// MessagesClient provides operations for Discord messages
type MessagesClient interface {
	// Send sends a message to a channel
	Send(ctx context.Context, channelID string, message *MessageCreate) (*Message, error)

	// Edit edits an existing message
	Edit(ctx context.Context, channelID, messageID string, message *MessageEdit) (*Message, error)

	// Delete deletes a message
	Delete(ctx context.Context, channelID, messageID string) error

	// Get retrieves a message
	Get(ctx context.Context, channelID, messageID string) (*Message, error)

	// GetHistory retrieves message history for a channel
	GetHistory(ctx context.Context, channelID string, params HistoryParams) ([]Message, error)

	// React adds a reaction to a message
	React(ctx context.Context, channelID, messageID, emoji string) error

	// DeleteReaction removes a reaction from a message
	DeleteReaction(ctx context.Context, channelID, messageID, emoji string) error
}

// ChannelsClient provides operations for Discord channels
type ChannelsClient interface {
	// Get retrieves a channel
	Get(ctx context.Context, channelID string) (*Channel, error)

	// Modify modifies a channel
	Modify(ctx context.Context, channelID string, params *ChannelModify) (*Channel, error)

	// Delete deletes a channel
	Delete(ctx context.Context, channelID string) error

	// CreateInvite creates an invite for a channel
	CreateInvite(ctx context.Context, channelID string, params *InviteCreate) (*Invite, error)
}

// GuildsClient provides operations for Discord guilds (servers)
type GuildsClient interface {
	// Get retrieves a guild
	Get(ctx context.Context, guildID string) (*Guild, error)

	// GetChannels retrieves channels for a guild
	GetChannels(ctx context.Context, guildID string) ([]Channel, error)

	// GetMembers retrieves members for a guild
	GetMembers(ctx context.Context, guildID string, params MemberListParams) ([]Member, error)

	// GetMember retrieves a specific member
	GetMember(ctx context.Context, guildID, userID string) (*Member, error)

	// CreateChannel creates a channel in a guild
	CreateChannel(ctx context.Context, guildID string, params *ChannelCreate) (*Channel, error)

	// GetRoles retrieves roles for a guild
	GetRoles(ctx context.Context, guildID string) ([]Role, error)
}

// UsersClient provides operations for Discord users
type UsersClient interface {
	// GetCurrentUser retrieves the current bot user
	GetCurrentUser(ctx context.Context) (*User, error)

	// GetUser retrieves a user by ID
	GetUser(ctx context.Context, userID string) (*User, error)

	// CreateDM creates a DM channel with a user
	CreateDM(ctx context.Context, userID string) (*Channel, error)
}

// Message represents a Discord message
type Message struct {
	ID              string       `json:"id"`
	ChannelID       string       `json:"channel_id"`
	GuildID         string       `json:"guild_id,omitempty"`
	Author          *User        `json:"author"`
	Content         string       `json:"content"`
	Timestamp       string       `json:"timestamp"`
	EditedTimestamp string       `json:"edited_timestamp,omitempty"`
	TTS             bool         `json:"tts"`
	MentionEveryone bool         `json:"mention_everyone"`
	Mentions        []User       `json:"mentions"`
	MentionRoles    []string     `json:"mention_roles"`
	Attachments     []Attachment `json:"attachments"`
	Embeds          []Embed      `json:"embeds"`
	Reactions       []Reaction   `json:"reactions,omitempty"`
	Pinned          bool         `json:"pinned"`
	Type            int          `json:"type"`
}

// MessageCreate represents parameters for creating a message
type MessageCreate struct {
	Content         string            `json:"content,omitempty"`
	TTS             bool              `json:"tts,omitempty"`
	Embeds          []Embed           `json:"embeds,omitempty"`
	AllowedMentions *AllowedMentions  `json:"allowed_mentions,omitempty"`
	Components      []Component       `json:"components,omitempty"`
	Files           []File            `json:"-"`
	MessageReference *MessageReference `json:"message_reference,omitempty"`
}

// MessageEdit represents parameters for editing a message
type MessageEdit struct {
	Content         string            `json:"content,omitempty"`
	Embeds          []Embed           `json:"embeds,omitempty"`
	AllowedMentions *AllowedMentions  `json:"allowed_mentions,omitempty"`
	Components      []Component       `json:"components,omitempty"`
}

// MessageReference represents a message reference (for replies)
type MessageReference struct {
	MessageID       string `json:"message_id,omitempty"`
	ChannelID       string `json:"channel_id,omitempty"`
	GuildID         string `json:"guild_id,omitempty"`
	FailIfNotExists bool   `json:"fail_if_not_exists,omitempty"`
}

// AllowedMentions controls what mentions are allowed
type AllowedMentions struct {
	Parse       []string `json:"parse,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Users       []string `json:"users,omitempty"`
	RepliedUser bool     `json:"replied_user,omitempty"`
}

// Embed represents a Discord embed
type Embed struct {
	Title       string          `json:"title,omitempty"`
	Type        string          `json:"type,omitempty"`
	Description string          `json:"description,omitempty"`
	URL         string          `json:"url,omitempty"`
	Timestamp   string          `json:"timestamp,omitempty"`
	Color       int             `json:"color,omitempty"`
	Footer      *EmbedFooter    `json:"footer,omitempty"`
	Image       *EmbedImage     `json:"image,omitempty"`
	Thumbnail   *EmbedThumbnail `json:"thumbnail,omitempty"`
	Author      *EmbedAuthor    `json:"author,omitempty"`
	Fields      []EmbedField    `json:"fields,omitempty"`
}

// EmbedFooter represents an embed footer
type EmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

// EmbedImage represents an embed image
type EmbedImage struct {
	URL string `json:"url"`
}

// EmbedThumbnail represents an embed thumbnail
type EmbedThumbnail struct {
	URL string `json:"url"`
}

// EmbedAuthor represents an embed author
type EmbedAuthor struct {
	Name    string `json:"name"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

// EmbedField represents an embed field
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// Attachment represents a message attachment
type Attachment struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type,omitempty"`
	Size        int    `json:"size"`
	URL         string `json:"url"`
	ProxyURL    string `json:"proxy_url"`
	Height      int    `json:"height,omitempty"`
	Width       int    `json:"width,omitempty"`
}

// File represents a file to upload
type File struct {
	Name        string
	ContentType string
	Data        []byte
}

// Reaction represents a message reaction
type Reaction struct {
	Count int   `json:"count"`
	Me    bool  `json:"me"`
	Emoji Emoji `json:"emoji"`
}

// Emoji represents a Discord emoji
type Emoji struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name"`
	Animated bool   `json:"animated,omitempty"`
}

// Component represents a message component
type Component struct {
	Type       int         `json:"type"`
	CustomID   string      `json:"custom_id,omitempty"`
	Style      int         `json:"style,omitempty"`
	Label      string      `json:"label,omitempty"`
	Emoji      *Emoji      `json:"emoji,omitempty"`
	URL        string      `json:"url,omitempty"`
	Disabled   bool        `json:"disabled,omitempty"`
	Components []Component `json:"components,omitempty"`
}

// Channel represents a Discord channel
type Channel struct {
	ID                   string `json:"id"`
	Type                 int    `json:"type"`
	GuildID              string `json:"guild_id,omitempty"`
	Position             int    `json:"position,omitempty"`
	Name                 string `json:"name,omitempty"`
	Topic                string `json:"topic,omitempty"`
	NSFW                 bool   `json:"nsfw,omitempty"`
	LastMessageID        string `json:"last_message_id,omitempty"`
	Bitrate              int    `json:"bitrate,omitempty"`
	UserLimit            int    `json:"user_limit,omitempty"`
	RateLimitPerUser     int    `json:"rate_limit_per_user,omitempty"`
	Recipients           []User `json:"recipients,omitempty"`
	ParentID             string `json:"parent_id,omitempty"`
	LastPinTimestamp     string `json:"last_pin_timestamp,omitempty"`
}

// ChannelCreate represents parameters for creating a channel
type ChannelCreate struct {
	Name                 string `json:"name"`
	Type                 int    `json:"type,omitempty"`
	Topic                string `json:"topic,omitempty"`
	Bitrate              int    `json:"bitrate,omitempty"`
	UserLimit            int    `json:"user_limit,omitempty"`
	RateLimitPerUser     int    `json:"rate_limit_per_user,omitempty"`
	Position             int    `json:"position,omitempty"`
	ParentID             string `json:"parent_id,omitempty"`
	NSFW                 bool   `json:"nsfw,omitempty"`
}

// ChannelModify represents parameters for modifying a channel
type ChannelModify struct {
	Name             string `json:"name,omitempty"`
	Type             int    `json:"type,omitempty"`
	Topic            string `json:"topic,omitempty"`
	NSFW             bool   `json:"nsfw,omitempty"`
	RateLimitPerUser int    `json:"rate_limit_per_user,omitempty"`
	Bitrate          int    `json:"bitrate,omitempty"`
	UserLimit        int    `json:"user_limit,omitempty"`
	ParentID         string `json:"parent_id,omitempty"`
}

// Guild represents a Discord guild (server)
type Guild struct {
	ID                       string    `json:"id"`
	Name                     string    `json:"name"`
	Icon                     string    `json:"icon,omitempty"`
	Splash                   string    `json:"splash,omitempty"`
	OwnerID                  string    `json:"owner_id"`
	Region                   string    `json:"region"`
	AFKChannelID             string    `json:"afk_channel_id,omitempty"`
	AFKTimeout               int       `json:"afk_timeout"`
	VerificationLevel        int       `json:"verification_level"`
	DefaultMessageNotifications int    `json:"default_message_notifications"`
	ExplicitContentFilter    int       `json:"explicit_content_filter"`
	Roles                    []Role    `json:"roles"`
	Emojis                   []Emoji   `json:"emojis"`
	Features                 []string  `json:"features"`
	MFALevel                 int       `json:"mfa_level"`
	SystemChannelID          string    `json:"system_channel_id,omitempty"`
	MemberCount              int       `json:"member_count,omitempty"`
	Description              string    `json:"description,omitempty"`
}

// Member represents a Discord guild member
type Member struct {
	User                       *User    `json:"user,omitempty"`
	Nick                       string   `json:"nick,omitempty"`
	Avatar                     string   `json:"avatar,omitempty"`
	Roles                      []string `json:"roles"`
	JoinedAt                   string   `json:"joined_at"`
	PremiumSince               string   `json:"premium_since,omitempty"`
	Deaf                       bool     `json:"deaf"`
	Mute                       bool     `json:"mute"`
	Pending                    bool     `json:"pending,omitempty"`
	CommunicationDisabledUntil string   `json:"communication_disabled_until,omitempty"`
}

// MemberListParams represents parameters for listing members
type MemberListParams struct {
	Limit int    `json:"limit,omitempty"`
	After string `json:"after,omitempty"`
}

// Role represents a Discord role
type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       int    `json:"color"`
	Hoist       bool   `json:"hoist"`
	Position    int    `json:"position"`
	Permissions string `json:"permissions"`
	Managed     bool   `json:"managed"`
	Mentionable bool   `json:"mentionable"`
}

// User represents a Discord user
type User struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	GlobalName    string `json:"global_name,omitempty"`
	Avatar        string `json:"avatar,omitempty"`
	Bot           bool   `json:"bot,omitempty"`
	System        bool   `json:"system,omitempty"`
	MFAEnabled    bool   `json:"mfa_enabled,omitempty"`
	Banner        string `json:"banner,omitempty"`
	AccentColor   int    `json:"accent_color,omitempty"`
	Locale        string `json:"locale,omitempty"`
	Verified      bool   `json:"verified,omitempty"`
	Email         string `json:"email,omitempty"`
	Flags         int    `json:"flags,omitempty"`
	PremiumType   int    `json:"premium_type,omitempty"`
	PublicFlags   int    `json:"public_flags,omitempty"`
}

// Invite represents a Discord invite
type Invite struct {
	Code      string   `json:"code"`
	Guild     *Guild   `json:"guild,omitempty"`
	Channel   *Channel `json:"channel"`
	Inviter   *User    `json:"inviter,omitempty"`
	Uses      int      `json:"uses,omitempty"`
	MaxUses   int      `json:"max_uses,omitempty"`
	MaxAge    int      `json:"max_age,omitempty"`
	Temporary bool     `json:"temporary,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
}

// InviteCreate represents parameters for creating an invite
type InviteCreate struct {
	MaxAge    int  `json:"max_age,omitempty"`
	MaxUses   int  `json:"max_uses,omitempty"`
	Temporary bool `json:"temporary,omitempty"`
	Unique    bool `json:"unique,omitempty"`
}

// HistoryParams represents parameters for getting message history
type HistoryParams struct {
	Limit  int    `json:"limit,omitempty"`
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
	Around string `json:"around,omitempty"`
}

// APIResponse represents a generic API response
type APIResponse struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}
