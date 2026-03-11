package discord

import (
	"cows/domain"
	"cows/usecases"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const (
	cmdDuel = "duel"
)

type Handler struct {
	appID   string
	service *usecases.Service
}

func New(appID string, service *usecases.Service) *Handler {
	return &Handler{appID: appID, service: service}
}

func (h *Handler) RegisterCommands(s *discordgo.Session, guildID string) error {
	duel := &discordgo.ApplicationCommand{
		Name:        cmdDuel,
		Description: "Start Bulls & Cows duel",
		Options: []*discordgo.ApplicationCommandOption{{
			Type:        discordgo.ApplicationCommandOptionUser,
			Name:        "opponent",
			Description: "Player to duel",
			Required:    true,
		}},
	}
	_, err := s.ApplicationCommandCreate(h.appID, guildID, duel)
	if err != nil {
		return err
	}
	userCmd := &discordgo.ApplicationCommand{Name: "Duel", Type: discordgo.UserApplicationCommand}
	_, err = s.ApplicationCommandCreate(h.appID, guildID, userCmd)
	return err
}

func (h *Handler) OnInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var err error
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		err = h.handleCommand(s, i)
	case discordgo.InteractionMessageComponent:
		err = h.handleComponent(s, i)
	case discordgo.InteractionModalSubmit:
		err = h.handleModal(s, i)
	}
	if err != nil {
		log.Printf("interaction failed: %v", err)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "Error: " + err.Error(), Flags: discordgo.MessageFlagsEphemeral}})
	}
}

func userID(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

func (h *Handler) handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	d := i.ApplicationCommandData()
	actor := userID(i)
	if d.Type == discordgo.UserApplicationCommand {
		if len(d.Resolved.Users) == 0 {
			return fmt.Errorf("user not resolved")
		}
		for _, u := range d.Resolved.Users {
			return h.respondChallenge(s, i, actor, u.ID)
		}
	}
	switch d.Name {
	case cmdDuel:
		op := d.Options[0].UserValue(s)
		if op == nil {
			return fmt.Errorf("opponent required")
		}
		return h.respondChallenge(s, i, actor, op.ID)
	default:
		return fmt.Errorf("unknown command")
	}
}

func (h *Handler) respondChallenge(s *discordgo.Session, i *discordgo.InteractionCreate, actor, opponent string) error {
	m, err := h.service.CreateChallenge(i.ID, actor, opponent)
	if err != nil {
		return err
	}
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{
		Content: fmt.Sprintf("<@%s> challenges <@%s> to Bulls & Cows!", actor, opponent),
		Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "Accept", Style: discordgo.SuccessButton, CustomID: "accept:" + m.ID},
			discordgo.Button{Label: "Decline", Style: discordgo.DangerButton, CustomID: "decline:" + m.ID},
		}}},
	}})
}

func (h *Handler) handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cid := i.MessageComponentData().CustomID
	parts := strings.Split(cid, ":")
	if len(parts) < 2 {
		return fmt.Errorf("invalid custom id")
	}
	actor := userID(i)
	matchID := parts[1]
	switch parts[0] {
	case "accept":
		_, err := h.service.AcceptChallenge(i.ID, matchID, actor)
		if err != nil {
			return err
		}
		return s.InteractionRespond(i.Interaction, modalResponse("secret_submit:"+matchID, "Enter secret code"))
	case "decline":
		_, err := h.service.DeclineChallenge(i.ID, matchID, actor)
		if err != nil {
			return err
		}
		return h.simpleUpdate(s, i, "Challenge declined.")
	case "secret_open":
		return s.InteractionRespond(i.Interaction, modalResponse("secret_submit:"+matchID, "Enter secret code"))
	case "secret_confirm":
		_, err := h.service.ConfirmSecret(i.ID, matchID, actor)
		if err != nil {
			return err
		}
		return h.simpleUpdate(s, i, "Secret confirmed.")
	case "secret_edit":
		_, err := h.service.EditSecret(i.ID, matchID, actor)
		if err != nil {
			return err
		}
		return s.InteractionRespond(i.Interaction, modalResponse("secret_submit:"+matchID, "Edit secret code"))
	case "cancel":
		_, err := h.service.CancelMatch(i.ID, matchID, actor)
		if err != nil {
			return err
		}
		return h.simpleUpdate(s, i, "Match cancelled.")
	case "guess_open":
		return s.InteractionRespond(i.Interaction, modalResponse("guess_submit:"+matchID, "Enter guess"))
	case "guess_confirm":
		m, err := h.service.ConfirmGuess(i.ID, matchID, actor)
		if err != nil {
			return err
		}
		return h.simpleUpdate(s, i, renderMatchProgress(m))
	case "rematch":
		m, err := h.service.CreateRematch(i.ID, matchID, actor)
		if err != nil {
			return err
		}
		return h.simpleUpdate(s, i, fmt.Sprintf("Rematch created: <@%s> vs <@%s>", m.Player1ID, m.Player2ID))
	default:
		return fmt.Errorf("unknown action")
	}
}

func modalResponse(customID, title string) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: &discordgo.InteractionResponseData{
		CustomID: customID,
		Title:    title,
		Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: "code", Label: "4 unique digits", Style: discordgo.TextInputShort, MinLength: 4, MaxLength: 4, Required: true},
		}}},
	}}
}

func (h *Handler) handleModal(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	data := i.ModalSubmitData()
	parts := strings.Split(data.CustomID, ":")
	if len(parts) < 2 {
		return fmt.Errorf("invalid modal")
	}
	matchID := parts[1]
	actor := userID(i)
	code := modalValue(data.Components, "code")
	switch parts[0] {
	case "secret_submit":
		_, err := h.service.SubmitSecret(i.ID, matchID, actor, code)
		if err != nil {
			return err
		}
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{
			Content: "Secret received. Confirm within 5 seconds or it will auto-confirm.",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "Confirm now", Style: discordgo.SuccessButton, CustomID: "secret_confirm:" + matchID},
				discordgo.Button{Label: "Edit", Style: discordgo.SecondaryButton, CustomID: "secret_edit:" + matchID},
				discordgo.Button{Label: "Cancel", Style: discordgo.DangerButton, CustomID: "cancel:" + matchID},
			}}},
		}})
	case "guess_submit":
		_, err := h.service.SubmitGuess(i.ID, matchID, actor, code)
		if err != nil {
			return err
		}
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{
			Content: "Guess received. Confirm within 5 seconds or it will auto-confirm.",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "Confirm now", Style: discordgo.SuccessButton, CustomID: "guess_confirm:" + matchID},
				discordgo.Button{Label: "Edit", Style: discordgo.SecondaryButton, CustomID: "guess_open:" + matchID},
				discordgo.Button{Label: "Cancel", Style: discordgo.DangerButton, CustomID: "cancel:" + matchID},
			}}},
		}})
	default:
		return fmt.Errorf("unknown modal")
	}
}

func modalValue(rows []discordgo.MessageComponent, id string) string {
	for _, r := range rows {
		if row, ok := r.(*discordgo.ActionsRow); ok {
			for _, c := range row.Components {
				if ti, ok := c.(*discordgo.TextInput); ok && ti.CustomID == id {
					return ti.Value
				}
			}
		}
	}
	return ""
}

func renderMatchProgress(m *domain.Match) string {
	if m.State == domain.StateFinished {
		return fmt.Sprintf("🏆 Winner: <@%s>", m.WinnerID)
	}
	if len(m.History) == 0 {
		return "No turns yet."
	}
	last := m.History[len(m.History)-1]
	return fmt.Sprintf("<@%s> guessed `%s` → %d bulls, %d cows.", last.PlayerID, last.Guess, last.Bulls, last.Cows)
}

func (h *Handler) simpleUpdate(s *discordgo.Session, i *discordgo.InteractionCreate, content string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Content: content, Components: []discordgo.MessageComponent{}}})
}
