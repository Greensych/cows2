package discordgo

type Session struct{}

type User struct{ ID string }
type Member struct{ User *User }

type MessageFlags int

const MessageFlagsEphemeral MessageFlags = 1 << 6

type InteractionType int

const (
	InteractionApplicationCommand InteractionType = 2
	InteractionMessageComponent   InteractionType = 3
	InteractionModalSubmit        InteractionType = 5
)

type ApplicationCommandType int

const (
	ChatApplicationCommand ApplicationCommandType = 1
	UserApplicationCommand ApplicationCommandType = 2
)

type ApplicationCommandOptionType int

const ApplicationCommandOptionUser ApplicationCommandOptionType = 6

type ButtonStyle int

const (
	PrimaryButton   ButtonStyle = 1
	SecondaryButton ButtonStyle = 2
	SuccessButton   ButtonStyle = 3
	DangerButton    ButtonStyle = 4
)

type TextInputStyle int

const TextInputShort TextInputStyle = 1

type InteractionResponseType int

const (
	InteractionResponseChannelMessageWithSource         InteractionResponseType = 4
	InteractionResponseDeferredChannelMessageWithSource InteractionResponseType = 5
	InteractionResponseDeferredMessageUpdate            InteractionResponseType = 6
	InteractionResponseUpdateMessage                    InteractionResponseType = 7
	InteractionResponseModal                            InteractionResponseType = 9
)

type ApplicationCommand struct {
	Name        string
	Description string
	Type        ApplicationCommandType
	Options     []*ApplicationCommandOption
}

type ApplicationCommandOption struct {
	Type        ApplicationCommandOptionType
	Name        string
	Description string
	Required    bool
}

func (o *ApplicationCommandInteractionDataOption) UserValue(_ *Session) *User { return o.user }

type ApplicationCommandInteractionDataOption struct {
	user *User
}

type ApplicationCommandInteractionDataResolved struct {
	Users map[string]*User
}

type ApplicationCommandInteractionData struct {
	Name     string
	Type     ApplicationCommandType
	Options  []*ApplicationCommandInteractionDataOption
	Resolved *ApplicationCommandInteractionDataResolved
}

type MessageComponentInteractionData struct{ CustomID string }

type ModalSubmitInteractionData struct {
	CustomID   string
	Components []MessageComponent
}

type Interaction struct{}

type InteractionCreate struct {
	Type        InteractionType
	ID          string
	Interaction *Interaction
	Member      *Member
	User        *User
	appData     ApplicationCommandInteractionData
	compData    MessageComponentInteractionData
	modalData   ModalSubmitInteractionData
}

func (i *InteractionCreate) ApplicationCommandData() ApplicationCommandInteractionData {
	return i.appData
}
func (i *InteractionCreate) MessageComponentData() MessageComponentInteractionData { return i.compData }
func (i *InteractionCreate) ModalSubmitData() ModalSubmitInteractionData           { return i.modalData }

type InteractionResponse struct {
	Type InteractionResponseType
	Data *InteractionResponseData
}

type InteractionResponseData struct {
	Content    string
	Flags      MessageFlags
	Components []MessageComponent
	CustomID   string
	Title      string
}

type MessageComponent interface{}

type ActionsRow struct{ Components []MessageComponent }
type Button struct {
	Label    string
	Style    ButtonStyle
	CustomID string
}
type TextInput struct {
	CustomID, Label      string
	Style                TextInputStyle
	MinLength, MaxLength int
	Required             bool
	Value                string
}

func New(_ string) (*Session, error)               { return &Session{}, nil }
func (s *Session) AddHandler(_ interface{}) func() { return func() {} }
func (s *Session) Open() error                     { return nil }
func (s *Session) Close() error                    { return nil }
func (s *Session) ApplicationCommandCreate(_, _ string, c *ApplicationCommand) (*ApplicationCommand, error) {
	return c, nil
}
func (s *Session) InteractionRespond(_ *Interaction, _ *InteractionResponse) error { return nil }
