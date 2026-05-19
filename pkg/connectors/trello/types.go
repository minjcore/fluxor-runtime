package trello

import "context"

// Client is the main Trello client interface
type Client interface {
	Boards() BoardsClient
	Lists() ListsClient
	Cards() CardsClient
	Members() MembersClient
}

// BoardsClient provides operations for Trello boards
type BoardsClient interface {
	Get(ctx context.Context, boardID string) (*Board, error)
	Create(ctx context.Context, params *CreateBoardParams) (*Board, error)
	Update(ctx context.Context, boardID string, params *UpdateBoardParams) (*Board, error)
	Delete(ctx context.Context, boardID string) error
	GetLists(ctx context.Context, boardID string) ([]List, error)
	GetCards(ctx context.Context, boardID string) ([]Card, error)
	GetMembers(ctx context.Context, boardID string) ([]Member, error)
}

// ListsClient provides operations for Trello lists
type ListsClient interface {
	Get(ctx context.Context, listID string) (*List, error)
	Create(ctx context.Context, params *CreateListParams) (*List, error)
	Update(ctx context.Context, listID string, params *UpdateListParams) (*List, error)
	Archive(ctx context.Context, listID string) (*List, error)
	GetCards(ctx context.Context, listID string) ([]Card, error)
	MoveAllCards(ctx context.Context, listID, targetListID string) error
}

// CardsClient provides operations for Trello cards
type CardsClient interface {
	Get(ctx context.Context, cardID string) (*Card, error)
	Create(ctx context.Context, params *CreateCardParams) (*Card, error)
	Update(ctx context.Context, cardID string, params *UpdateCardParams) (*Card, error)
	Delete(ctx context.Context, cardID string) error
	Archive(ctx context.Context, cardID string) (*Card, error)
	Move(ctx context.Context, cardID, listID string) (*Card, error)
	AddComment(ctx context.Context, cardID, text string) (*Action, error)
	AddLabel(ctx context.Context, cardID, labelID string) error
	RemoveLabel(ctx context.Context, cardID, labelID string) error
	AddMember(ctx context.Context, cardID, memberID string) error
	RemoveMember(ctx context.Context, cardID, memberID string) error
	GetChecklists(ctx context.Context, cardID string) ([]Checklist, error)
	AddChecklist(ctx context.Context, cardID string, params *CreateChecklistParams) (*Checklist, error)
}

// MembersClient provides operations for Trello members
type MembersClient interface {
	Get(ctx context.Context, memberID string) (*Member, error)
	GetMe(ctx context.Context) (*Member, error)
	GetBoards(ctx context.Context, memberID string) ([]Board, error)
}

// Board represents a Trello board
type Board struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Desc           string            `json:"desc"`
	DescData       string            `json:"descData,omitempty"`
	Closed         bool              `json:"closed"`
	IDOrganization string            `json:"idOrganization,omitempty"`
	Pinned         bool              `json:"pinned"`
	URL            string            `json:"url"`
	ShortURL       string            `json:"shortUrl"`
	Prefs          *BoardPrefs       `json:"prefs,omitempty"`
	LabelNames     map[string]string `json:"labelNames,omitempty"`
	Starred        bool              `json:"starred"`
	Memberships    []Membership      `json:"memberships,omitempty"`
	ShortLink      string            `json:"shortLink"`
	DateLastActivity string          `json:"dateLastActivity,omitempty"`
	DateLastView   string            `json:"dateLastView,omitempty"`
}

// BoardPrefs represents board preferences
type BoardPrefs struct {
	PermissionLevel       string `json:"permissionLevel"`
	Voting                string `json:"voting"`
	Comments              string `json:"comments"`
	Invitations           string `json:"invitations"`
	SelfJoin              bool   `json:"selfJoin"`
	CardCovers            bool   `json:"cardCovers"`
	CardAging             string `json:"cardAging"`
	CalendarFeedEnabled   bool   `json:"calendarFeedEnabled"`
	Background            string `json:"background"`
	BackgroundColor       string `json:"backgroundColor,omitempty"`
	BackgroundImage       string `json:"backgroundImage,omitempty"`
	BackgroundImageScaled []struct {
		Width  int    `json:"width"`
		Height int    `json:"height"`
		URL    string `json:"url"`
	} `json:"backgroundImageScaled,omitempty"`
	BackgroundTile        bool   `json:"backgroundTile"`
	BackgroundBrightness  string `json:"backgroundBrightness"`
	HideVotes             bool   `json:"hideVotes"`
}

// CreateBoardParams represents parameters for creating a board
type CreateBoardParams struct {
	Name           string `json:"name"`
	Desc           string `json:"desc,omitempty"`
	IDOrganization string `json:"idOrganization,omitempty"`
	DefaultLists   bool   `json:"defaultLists,omitempty"`
	DefaultLabels  bool   `json:"defaultLabels,omitempty"`
	Prefs          *BoardPrefsParams `json:"prefs,omitempty"`
}

// BoardPrefsParams represents board preferences parameters
type BoardPrefsParams struct {
	PermissionLevel string `json:"permissionLevel,omitempty"`
	Voting          string `json:"voting,omitempty"`
	Comments        string `json:"comments,omitempty"`
	Background      string `json:"background,omitempty"`
}

// UpdateBoardParams represents parameters for updating a board
type UpdateBoardParams struct {
	Name   string `json:"name,omitempty"`
	Desc   string `json:"desc,omitempty"`
	Closed *bool  `json:"closed,omitempty"`
	Prefs  *BoardPrefsParams `json:"prefs,omitempty"`
}

// List represents a Trello list
type List struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Closed   bool   `json:"closed"`
	IDBoard  string `json:"idBoard"`
	Pos      float64 `json:"pos"`
	Subscribed bool  `json:"subscribed"`
}

// CreateListParams represents parameters for creating a list
type CreateListParams struct {
	Name    string  `json:"name"`
	IDBoard string  `json:"idBoard"`
	Pos     string  `json:"pos,omitempty"` // "top", "bottom", or a positive number
}

// UpdateListParams represents parameters for updating a list
type UpdateListParams struct {
	Name   string `json:"name,omitempty"`
	Closed *bool  `json:"closed,omitempty"`
	Pos    string `json:"pos,omitempty"`
}

// Card represents a Trello card
type Card struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Desc              string   `json:"desc"`
	Closed            bool     `json:"closed"`
	IDBoard           string   `json:"idBoard"`
	IDList            string   `json:"idList"`
	IDMembers         []string `json:"idMembers"`
	IDLabels          []string `json:"idLabels"`
	IDChecklists      []string `json:"idChecklists"`
	Pos               float64  `json:"pos"`
	Due               string   `json:"due,omitempty"`
	DueComplete       bool     `json:"dueComplete"`
	URL               string   `json:"url"`
	ShortURL          string   `json:"shortUrl"`
	ShortLink         string   `json:"shortLink"`
	Labels            []Label  `json:"labels,omitempty"`
	Subscribed        bool     `json:"subscribed"`
	DateLastActivity  string   `json:"dateLastActivity"`
	Start             string   `json:"start,omitempty"`
	Cover             *Cover   `json:"cover,omitempty"`
	Badges            *Badges  `json:"badges,omitempty"`
}

// Cover represents card cover settings
type Cover struct {
	IDAttachment         string `json:"idAttachment,omitempty"`
	Color                string `json:"color,omitempty"`
	IDUploadedBackground string `json:"idUploadedBackground,omitempty"`
	Size                 string `json:"size,omitempty"`
	Brightness           string `json:"brightness,omitempty"`
}

// Badges represents card badges
type Badges struct {
	Votes              int    `json:"votes"`
	ViewingMemberVoted bool   `json:"viewingMemberVoted"`
	Subscribed         bool   `json:"subscribed"`
	Fogbugz            string `json:"fogbugz"`
	CheckItems         int    `json:"checkItems"`
	CheckItemsChecked  int    `json:"checkItemsChecked"`
	Comments           int    `json:"comments"`
	Attachments        int    `json:"attachments"`
	Description        bool   `json:"description"`
	Due                string `json:"due,omitempty"`
	DueComplete        bool   `json:"dueComplete"`
	Start              string `json:"start,omitempty"`
}

// CreateCardParams represents parameters for creating a card
type CreateCardParams struct {
	Name      string   `json:"name"`
	Desc      string   `json:"desc,omitempty"`
	IDList    string   `json:"idList"`
	IDMembers []string `json:"idMembers,omitempty"`
	IDLabels  []string `json:"idLabels,omitempty"`
	Pos       string   `json:"pos,omitempty"`
	Due       string   `json:"due,omitempty"`
	Start     string   `json:"start,omitempty"`
}

// UpdateCardParams represents parameters for updating a card
type UpdateCardParams struct {
	Name        string   `json:"name,omitempty"`
	Desc        string   `json:"desc,omitempty"`
	Closed      *bool    `json:"closed,omitempty"`
	IDList      string   `json:"idList,omitempty"`
	IDMembers   []string `json:"idMembers,omitempty"`
	IDLabels    []string `json:"idLabels,omitempty"`
	Pos         string   `json:"pos,omitempty"`
	Due         string   `json:"due,omitempty"`
	DueComplete *bool    `json:"dueComplete,omitempty"`
	Start       string   `json:"start,omitempty"`
}

// Label represents a Trello label
type Label struct {
	ID      string `json:"id"`
	IDBoard string `json:"idBoard"`
	Name    string `json:"name"`
	Color   string `json:"color"`
}

// Member represents a Trello member
type Member struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	FullName   string `json:"fullName"`
	Initials   string `json:"initials"`
	AvatarHash string `json:"avatarHash,omitempty"`
	AvatarURL  string `json:"avatarUrl,omitempty"`
	Email      string `json:"email,omitempty"`
	URL        string `json:"url"`
	Bio        string `json:"bio,omitempty"`
	MemberType string `json:"memberType,omitempty"`
	Status     string `json:"status,omitempty"`
	Confirmed  bool   `json:"confirmed"`
}

// Membership represents a board membership
type Membership struct {
	ID          string `json:"id"`
	IDMember    string `json:"idMember"`
	MemberType  string `json:"memberType"`
	Unconfirmed bool   `json:"unconfirmed"`
	Deactivated bool   `json:"deactivated"`
}

// Checklist represents a Trello checklist
type Checklist struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	IDBoard    string       `json:"idBoard"`
	IDCard     string       `json:"idCard"`
	Pos        float64      `json:"pos"`
	CheckItems []CheckItem  `json:"checkItems"`
}

// CheckItem represents a checklist item
type CheckItem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	State       string  `json:"state"` // "complete" or "incomplete"
	IDChecklist string  `json:"idChecklist"`
	Pos         float64 `json:"pos"`
	Due         string  `json:"due,omitempty"`
	IDMember    string  `json:"idMember,omitempty"`
}

// CreateChecklistParams represents parameters for creating a checklist
type CreateChecklistParams struct {
	Name        string `json:"name"`
	IDChecklistSource string `json:"idChecklistSource,omitempty"`
	Pos         string `json:"pos,omitempty"`
}

// Action represents a Trello action (e.g., comment)
type Action struct {
	ID              string      `json:"id"`
	IDMemberCreator string      `json:"idMemberCreator"`
	Type            string      `json:"type"`
	Date            string      `json:"date"`
	Data            *ActionData `json:"data,omitempty"`
	MemberCreator   *Member     `json:"memberCreator,omitempty"`
}

// ActionData represents action data
type ActionData struct {
	Text  string `json:"text,omitempty"`
	Board *struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		ShortLink string `json:"shortLink"`
	} `json:"board,omitempty"`
	Card *struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		ShortLink string `json:"shortLink"`
	} `json:"card,omitempty"`
	List *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"list,omitempty"`
}

// APIError represents a Trello API error
type APIError struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}
