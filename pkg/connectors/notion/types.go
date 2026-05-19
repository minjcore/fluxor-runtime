package notion

import "context"

// Client is the main Notion client interface
type Client interface {
	Pages() PagesClient
	Databases() DatabasesClient
	Blocks() BlocksClient
	Users() UsersClient
	Search() SearchClient
}

// PagesClient provides operations for Notion pages
type PagesClient interface {
	// Create creates a new page
	Create(ctx context.Context, req *CreatePageRequest) (*Page, error)

	// Get retrieves a page
	Get(ctx context.Context, pageID string) (*Page, error)

	// Update updates a page's properties
	Update(ctx context.Context, pageID string, properties map[string]Property) (*Page, error)

	// Archive archives a page
	Archive(ctx context.Context, pageID string) (*Page, error)
}

// DatabasesClient provides operations for Notion databases
type DatabasesClient interface {
	// Create creates a new database
	Create(ctx context.Context, req *CreateDatabaseRequest) (*Database, error)

	// Get retrieves a database
	Get(ctx context.Context, databaseID string) (*Database, error)

	// Query queries a database
	Query(ctx context.Context, databaseID string, query *DatabaseQuery) (*QueryResult, error)

	// Update updates a database
	Update(ctx context.Context, databaseID string, req *UpdateDatabaseRequest) (*Database, error)
}

// BlocksClient provides operations for Notion blocks
type BlocksClient interface {
	// Get retrieves a block
	Get(ctx context.Context, blockID string) (*Block, error)

	// GetChildren retrieves a block's children
	GetChildren(ctx context.Context, blockID string, cursor string) (*BlockChildrenResponse, error)

	// AppendChildren appends children to a block
	AppendChildren(ctx context.Context, blockID string, children []Block) (*BlockChildrenResponse, error)

	// Update updates a block
	Update(ctx context.Context, blockID string, block *Block) (*Block, error)

	// Delete deletes a block
	Delete(ctx context.Context, blockID string) (*Block, error)
}

// UsersClient provides operations for Notion users
type UsersClient interface {
	// Get retrieves a user
	Get(ctx context.Context, userID string) (*User, error)

	// List lists all users
	List(ctx context.Context, cursor string) (*UsersResponse, error)

	// GetMe retrieves the bot user
	GetMe(ctx context.Context) (*User, error)
}

// SearchClient provides search operations
type SearchClient interface {
	// Search searches pages and databases
	Search(ctx context.Context, query *SearchRequest) (*SearchResponse, error)
}

// Page represents a Notion page
type Page struct {
	Object         string              `json:"object"`
	ID             string              `json:"id"`
	CreatedTime    string              `json:"created_time"`
	LastEditedTime string              `json:"last_edited_time"`
	CreatedBy      *User               `json:"created_by,omitempty"`
	LastEditedBy   *User               `json:"last_edited_by,omitempty"`
	Cover          *File               `json:"cover,omitempty"`
	Icon           *Icon               `json:"icon,omitempty"`
	Parent         Parent              `json:"parent"`
	Archived       bool                `json:"archived"`
	Properties     map[string]Property `json:"properties"`
	URL            string              `json:"url"`
}

// CreatePageRequest represents a request to create a page
type CreatePageRequest struct {
	Parent     Parent              `json:"parent"`
	Properties map[string]Property `json:"properties"`
	Children   []Block             `json:"children,omitempty"`
	Icon       *Icon               `json:"icon,omitempty"`
	Cover      *File               `json:"cover,omitempty"`
}

// Database represents a Notion database
type Database struct {
	Object         string                      `json:"object"`
	ID             string                      `json:"id"`
	CreatedTime    string                      `json:"created_time"`
	LastEditedTime string                      `json:"last_edited_time"`
	CreatedBy      *User                       `json:"created_by,omitempty"`
	LastEditedBy   *User                       `json:"last_edited_by,omitempty"`
	Title          []RichText                  `json:"title"`
	Description    []RichText                  `json:"description"`
	Icon           *Icon                       `json:"icon,omitempty"`
	Cover          *File                       `json:"cover,omitempty"`
	Properties     map[string]PropertyConfig   `json:"properties"`
	Parent         Parent                      `json:"parent"`
	URL            string                      `json:"url"`
	Archived       bool                        `json:"archived"`
	IsInline       bool                        `json:"is_inline"`
}

// CreateDatabaseRequest represents a request to create a database
type CreateDatabaseRequest struct {
	Parent     Parent                    `json:"parent"`
	Title      []RichText                `json:"title"`
	Properties map[string]PropertyConfig `json:"properties"`
	Icon       *Icon                     `json:"icon,omitempty"`
	Cover      *File                     `json:"cover,omitempty"`
}

// UpdateDatabaseRequest represents a request to update a database
type UpdateDatabaseRequest struct {
	Title       []RichText                `json:"title,omitempty"`
	Description []RichText                `json:"description,omitempty"`
	Properties  map[string]PropertyConfig `json:"properties,omitempty"`
	Icon        *Icon                     `json:"icon,omitempty"`
	Cover       *File                     `json:"cover,omitempty"`
}

// DatabaseQuery represents a database query
type DatabaseQuery struct {
	Filter      *Filter   `json:"filter,omitempty"`
	Sorts       []Sort    `json:"sorts,omitempty"`
	StartCursor string    `json:"start_cursor,omitempty"`
	PageSize    int       `json:"page_size,omitempty"`
}

// QueryResult represents the result of a database query
type QueryResult struct {
	Object     string `json:"object"`
	Results    []Page `json:"results"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// Filter represents a database filter
type Filter struct {
	// Property filter
	Property string      `json:"property,omitempty"`
	RichText *TextFilter `json:"rich_text,omitempty"`
	Title    *TextFilter `json:"title,omitempty"`
	Number   *NumberFilter `json:"number,omitempty"`
	Checkbox *CheckboxFilter `json:"checkbox,omitempty"`
	Select   *SelectFilter `json:"select,omitempty"`
	Date     *DateFilter `json:"date,omitempty"`
	
	// Compound filters
	Or  []Filter `json:"or,omitempty"`
	And []Filter `json:"and,omitempty"`
}

// TextFilter represents a text filter
type TextFilter struct {
	Equals         string `json:"equals,omitempty"`
	DoesNotEqual   string `json:"does_not_equal,omitempty"`
	Contains       string `json:"contains,omitempty"`
	DoesNotContain string `json:"does_not_contain,omitempty"`
	StartsWith     string `json:"starts_with,omitempty"`
	EndsWith       string `json:"ends_with,omitempty"`
	IsEmpty        bool   `json:"is_empty,omitempty"`
	IsNotEmpty     bool   `json:"is_not_empty,omitempty"`
}

// NumberFilter represents a number filter
type NumberFilter struct {
	Equals               *float64 `json:"equals,omitempty"`
	DoesNotEqual         *float64 `json:"does_not_equal,omitempty"`
	GreaterThan          *float64 `json:"greater_than,omitempty"`
	LessThan             *float64 `json:"less_than,omitempty"`
	GreaterThanOrEqualTo *float64 `json:"greater_than_or_equal_to,omitempty"`
	LessThanOrEqualTo    *float64 `json:"less_than_or_equal_to,omitempty"`
	IsEmpty              bool     `json:"is_empty,omitempty"`
	IsNotEmpty           bool     `json:"is_not_empty,omitempty"`
}

// CheckboxFilter represents a checkbox filter
type CheckboxFilter struct {
	Equals       *bool `json:"equals,omitempty"`
	DoesNotEqual *bool `json:"does_not_equal,omitempty"`
}

// SelectFilter represents a select filter
type SelectFilter struct {
	Equals       string `json:"equals,omitempty"`
	DoesNotEqual string `json:"does_not_equal,omitempty"`
	IsEmpty      bool   `json:"is_empty,omitempty"`
	IsNotEmpty   bool   `json:"is_not_empty,omitempty"`
}

// DateFilter represents a date filter
type DateFilter struct {
	Equals     string `json:"equals,omitempty"`
	Before     string `json:"before,omitempty"`
	After      string `json:"after,omitempty"`
	OnOrBefore string `json:"on_or_before,omitempty"`
	OnOrAfter  string `json:"on_or_after,omitempty"`
	IsEmpty    bool   `json:"is_empty,omitempty"`
	IsNotEmpty bool   `json:"is_not_empty,omitempty"`
	PastWeek   *struct{} `json:"past_week,omitempty"`
	PastMonth  *struct{} `json:"past_month,omitempty"`
	PastYear   *struct{} `json:"past_year,omitempty"`
	NextWeek   *struct{} `json:"next_week,omitempty"`
	NextMonth  *struct{} `json:"next_month,omitempty"`
	NextYear   *struct{} `json:"next_year,omitempty"`
}

// Sort represents a database sort
type Sort struct {
	Property  string `json:"property,omitempty"`
	Timestamp string `json:"timestamp,omitempty"` // created_time or last_edited_time
	Direction string `json:"direction"`           // ascending or descending
}

// Block represents a Notion block
type Block struct {
	Object         string      `json:"object"`
	ID             string      `json:"id,omitempty"`
	Parent         *Parent     `json:"parent,omitempty"`
	Type           string      `json:"type"`
	CreatedTime    string      `json:"created_time,omitempty"`
	LastEditedTime string      `json:"last_edited_time,omitempty"`
	CreatedBy      *User       `json:"created_by,omitempty"`
	LastEditedBy   *User       `json:"last_edited_by,omitempty"`
	HasChildren    bool        `json:"has_children,omitempty"`
	Archived       bool        `json:"archived,omitempty"`
	
	// Block type specific content
	Paragraph        *ParagraphBlock        `json:"paragraph,omitempty"`
	Heading1         *HeadingBlock          `json:"heading_1,omitempty"`
	Heading2         *HeadingBlock          `json:"heading_2,omitempty"`
	Heading3         *HeadingBlock          `json:"heading_3,omitempty"`
	BulletedListItem *ListItemBlock         `json:"bulleted_list_item,omitempty"`
	NumberedListItem *ListItemBlock         `json:"numbered_list_item,omitempty"`
	ToDo             *ToDoBlock             `json:"to_do,omitempty"`
	Toggle           *ToggleBlock           `json:"toggle,omitempty"`
	Code             *CodeBlock             `json:"code,omitempty"`
	Quote            *QuoteBlock            `json:"quote,omitempty"`
	Callout          *CalloutBlock          `json:"callout,omitempty"`
	Divider          *struct{}              `json:"divider,omitempty"`
	Image            *FileBlock             `json:"image,omitempty"`
	Video            *FileBlock             `json:"video,omitempty"`
	File             *FileBlock             `json:"file,omitempty"`
	Bookmark         *BookmarkBlock         `json:"bookmark,omitempty"`
	Embed            *EmbedBlock            `json:"embed,omitempty"`
}

// ParagraphBlock represents a paragraph block
type ParagraphBlock struct {
	RichText []RichText `json:"rich_text"`
	Color    string     `json:"color,omitempty"`
	Children []Block    `json:"children,omitempty"`
}

// HeadingBlock represents a heading block
type HeadingBlock struct {
	RichText     []RichText `json:"rich_text"`
	Color        string     `json:"color,omitempty"`
	IsToggleable bool       `json:"is_toggleable,omitempty"`
}

// ListItemBlock represents a list item block
type ListItemBlock struct {
	RichText []RichText `json:"rich_text"`
	Color    string     `json:"color,omitempty"`
	Children []Block    `json:"children,omitempty"`
}

// ToDoBlock represents a to-do block
type ToDoBlock struct {
	RichText []RichText `json:"rich_text"`
	Checked  bool       `json:"checked"`
	Color    string     `json:"color,omitempty"`
	Children []Block    `json:"children,omitempty"`
}

// ToggleBlock represents a toggle block
type ToggleBlock struct {
	RichText []RichText `json:"rich_text"`
	Color    string     `json:"color,omitempty"`
	Children []Block    `json:"children,omitempty"`
}

// CodeBlock represents a code block
type CodeBlock struct {
	RichText []RichText `json:"rich_text"`
	Caption  []RichText `json:"caption,omitempty"`
	Language string     `json:"language"`
}

// QuoteBlock represents a quote block
type QuoteBlock struct {
	RichText []RichText `json:"rich_text"`
	Color    string     `json:"color,omitempty"`
	Children []Block    `json:"children,omitempty"`
}

// CalloutBlock represents a callout block
type CalloutBlock struct {
	RichText []RichText `json:"rich_text"`
	Icon     *Icon      `json:"icon,omitempty"`
	Color    string     `json:"color,omitempty"`
	Children []Block    `json:"children,omitempty"`
}

// FileBlock represents a file/image/video block
type FileBlock struct {
	Caption  []RichText `json:"caption,omitempty"`
	Type     string     `json:"type"` // external or file
	External *External  `json:"external,omitempty"`
	File     *FileInfo  `json:"file,omitempty"`
}

// BookmarkBlock represents a bookmark block
type BookmarkBlock struct {
	Caption []RichText `json:"caption,omitempty"`
	URL     string     `json:"url"`
}

// EmbedBlock represents an embed block
type EmbedBlock struct {
	URL string `json:"url"`
}

// BlockChildrenResponse represents the response for block children
type BlockChildrenResponse struct {
	Object     string  `json:"object"`
	Results    []Block `json:"results"`
	NextCursor string  `json:"next_cursor,omitempty"`
	HasMore    bool    `json:"has_more"`
}

// User represents a Notion user
type User struct {
	Object    string  `json:"object"`
	ID        string  `json:"id"`
	Type      string  `json:"type,omitempty"` // person or bot
	Name      string  `json:"name,omitempty"`
	AvatarURL string  `json:"avatar_url,omitempty"`
	Person    *Person `json:"person,omitempty"`
	Bot       *Bot    `json:"bot,omitempty"`
}

// Person represents a person user
type Person struct {
	Email string `json:"email,omitempty"`
}

// Bot represents a bot user
type Bot struct {
	Owner         *BotOwner `json:"owner,omitempty"`
	WorkspaceName string    `json:"workspace_name,omitempty"`
}

// BotOwner represents the owner of a bot
type BotOwner struct {
	Type      string `json:"type"` // workspace or user
	Workspace bool   `json:"workspace,omitempty"`
	User      *User  `json:"user,omitempty"`
}

// UsersResponse represents the response for listing users
type UsersResponse struct {
	Object     string `json:"object"`
	Results    []User `json:"results"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// Parent represents a parent object
type Parent struct {
	Type       string `json:"type"` // database_id, page_id, workspace, block_id
	DatabaseID string `json:"database_id,omitempty"`
	PageID     string `json:"page_id,omitempty"`
	Workspace  bool   `json:"workspace,omitempty"`
	BlockID    string `json:"block_id,omitempty"`
}

// Property represents a page property value
type Property struct {
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type"`
	Title    []RichText   `json:"title,omitempty"`
	RichText []RichText   `json:"rich_text,omitempty"`
	Number   *float64     `json:"number,omitempty"`
	Select   *SelectOption `json:"select,omitempty"`
	MultiSelect []SelectOption `json:"multi_select,omitempty"`
	Date     *DateProperty `json:"date,omitempty"`
	Checkbox *bool        `json:"checkbox,omitempty"`
	URL      string       `json:"url,omitempty"`
	Email    string       `json:"email,omitempty"`
	Phone    string       `json:"phone_number,omitempty"`
	People   []User       `json:"people,omitempty"`
	Files    []File       `json:"files,omitempty"`
	Relation []Relation   `json:"relation,omitempty"`
	Formula  *Formula     `json:"formula,omitempty"`
	Rollup   *Rollup      `json:"rollup,omitempty"`
	Status   *StatusOption `json:"status,omitempty"`
}

// PropertyConfig represents a database property configuration
type PropertyConfig struct {
	ID          string         `json:"id,omitempty"`
	Type        string         `json:"type"`
	Name        string         `json:"name,omitempty"`
	Title       *struct{}      `json:"title,omitempty"`
	RichText    *struct{}      `json:"rich_text,omitempty"`
	Number      *NumberConfig  `json:"number,omitempty"`
	Select      *SelectConfig  `json:"select,omitempty"`
	MultiSelect *SelectConfig  `json:"multi_select,omitempty"`
	Date        *struct{}      `json:"date,omitempty"`
	Checkbox    *struct{}      `json:"checkbox,omitempty"`
	URL         *struct{}      `json:"url,omitempty"`
	Email       *struct{}      `json:"email,omitempty"`
	Phone       *struct{}      `json:"phone_number,omitempty"`
	People      *struct{}      `json:"people,omitempty"`
	Files       *struct{}      `json:"files,omitempty"`
	Relation    *RelationConfig `json:"relation,omitempty"`
	Formula     *FormulaConfig  `json:"formula,omitempty"`
	Rollup      *RollupConfig   `json:"rollup,omitempty"`
	Status      *StatusConfig   `json:"status,omitempty"`
}

// NumberConfig represents number property configuration
type NumberConfig struct {
	Format string `json:"format,omitempty"`
}

// SelectConfig represents select property configuration
type SelectConfig struct {
	Options []SelectOption `json:"options,omitempty"`
}

// SelectOption represents a select option
type SelectOption struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// StatusOption represents a status option
type StatusOption struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// StatusConfig represents status property configuration
type StatusConfig struct {
	Options []StatusOption `json:"options,omitempty"`
	Groups  []StatusGroup  `json:"groups,omitempty"`
}

// StatusGroup represents a status group
type StatusGroup struct {
	ID        string   `json:"id,omitempty"`
	Name      string   `json:"name"`
	Color     string   `json:"color,omitempty"`
	OptionIDs []string `json:"option_ids,omitempty"`
}

// RelationConfig represents relation property configuration
type RelationConfig struct {
	DatabaseID         string `json:"database_id"`
	Type               string `json:"type,omitempty"`
	SingleProperty     *struct{} `json:"single_property,omitempty"`
	DualProperty       *DualProperty `json:"dual_property,omitempty"`
}

// DualProperty represents dual property configuration
type DualProperty struct {
	SyncedPropertyName string `json:"synced_property_name,omitempty"`
	SyncedPropertyID   string `json:"synced_property_id,omitempty"`
}

// FormulaConfig represents formula property configuration
type FormulaConfig struct {
	Expression string `json:"expression"`
}

// RollupConfig represents rollup property configuration
type RollupConfig struct {
	RelationPropertyName string `json:"relation_property_name"`
	RelationPropertyID   string `json:"relation_property_id"`
	RollupPropertyName   string `json:"rollup_property_name"`
	RollupPropertyID     string `json:"rollup_property_id"`
	Function             string `json:"function"`
}

// DateProperty represents a date property value
type DateProperty struct {
	Start    string `json:"start"`
	End      string `json:"end,omitempty"`
	TimeZone string `json:"time_zone,omitempty"`
}

// Relation represents a relation property value
type Relation struct {
	ID string `json:"id"`
}

// Formula represents a formula property value
type Formula struct {
	Type    string   `json:"type"`
	String  string   `json:"string,omitempty"`
	Number  *float64 `json:"number,omitempty"`
	Boolean *bool    `json:"boolean,omitempty"`
	Date    *DateProperty `json:"date,omitempty"`
}

// Rollup represents a rollup property value
type Rollup struct {
	Type   string      `json:"type"`
	Number *float64    `json:"number,omitempty"`
	Date   *DateProperty `json:"date,omitempty"`
	Array  []Property  `json:"array,omitempty"`
}

// RichText represents rich text content
type RichText struct {
	Type        string       `json:"type"`
	Text        *Text        `json:"text,omitempty"`
	Mention     *Mention     `json:"mention,omitempty"`
	Equation    *Equation    `json:"equation,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
	PlainText   string       `json:"plain_text,omitempty"`
	Href        string       `json:"href,omitempty"`
}

// Text represents text content
type Text struct {
	Content string `json:"content"`
	Link    *Link  `json:"link,omitempty"`
}

// Link represents a link
type Link struct {
	URL string `json:"url"`
}

// Mention represents a mention
type Mention struct {
	Type string `json:"type"` // user, page, database, date, link_preview
	User *User  `json:"user,omitempty"`
	Page *struct{ ID string } `json:"page,omitempty"`
	Database *struct{ ID string } `json:"database,omitempty"`
	Date *DateProperty `json:"date,omitempty"`
	LinkPreview *struct{ URL string } `json:"link_preview,omitempty"`
}

// Equation represents an equation
type Equation struct {
	Expression string `json:"expression"`
}

// Annotations represents text annotations
type Annotations struct {
	Bold          bool   `json:"bold"`
	Italic        bool   `json:"italic"`
	Strikethrough bool   `json:"strikethrough"`
	Underline     bool   `json:"underline"`
	Code          bool   `json:"code"`
	Color         string `json:"color"`
}

// Icon represents an icon
type Icon struct {
	Type     string    `json:"type"` // emoji or external or file
	Emoji    string    `json:"emoji,omitempty"`
	External *External `json:"external,omitempty"`
	File     *FileInfo `json:"file,omitempty"`
}

// File represents a file
type File struct {
	Type     string    `json:"type"` // external or file
	Name     string    `json:"name,omitempty"`
	External *External `json:"external,omitempty"`
	File     *FileInfo `json:"file,omitempty"`
}

// External represents an external file
type External struct {
	URL string `json:"url"`
}

// FileInfo represents file information
type FileInfo struct {
	URL        string `json:"url"`
	ExpiryTime string `json:"expiry_time,omitempty"`
}

// SearchRequest represents a search request
type SearchRequest struct {
	Query       string       `json:"query,omitempty"`
	Filter      *SearchFilter `json:"filter,omitempty"`
	Sort        *SearchSort  `json:"sort,omitempty"`
	StartCursor string       `json:"start_cursor,omitempty"`
	PageSize    int          `json:"page_size,omitempty"`
}

// SearchFilter represents a search filter
type SearchFilter struct {
	Value    string `json:"value"`    // page or database
	Property string `json:"property"` // object
}

// SearchSort represents a search sort
type SearchSort struct {
	Direction string `json:"direction"` // ascending or descending
	Timestamp string `json:"timestamp"` // last_edited_time
}

// SearchResponse represents a search response
type SearchResponse struct {
	Object     string        `json:"object"`
	Results    []interface{} `json:"results"` // Can be Page or Database
	NextCursor string        `json:"next_cursor,omitempty"`
	HasMore    bool          `json:"has_more"`
}

// APIResponse represents the Notion API response for errors
type APIResponse struct {
	Object  string `json:"object"`
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
