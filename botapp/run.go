package botapp

import (
	"cows/scheduler"
	"cows/store/memory"
	discordtransport "cows/transport/discord"
	"cows/usecases"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

func Run() {
	if err := loadDotEnv(".env"); err != nil {
		log.Fatalf("load .env: %v", err)
	}

	token := os.Getenv("DISCORD_TOKEN")
	appID := os.Getenv("DISCORD_APP_ID")
	guildID := os.Getenv("DISCORD_GUILD_ID")
	if token == "" || appID == "" {
		log.Fatal("set DISCORD_TOKEN and DISCORD_APP_ID (can be placed in .env)")
	}

	store := memory.New()
	sched := scheduler.NewMemoryScheduler()
	locker := usecases.NewKeyedLocker()
	svc := usecases.NewService(store, sched, locker, usecases.Durations{
		ChallengeTimeout: 60 * time.Second,
		SecretTimeout:    120 * time.Second,
		ConfirmTimeout:   5 * time.Second,
		TurnTimeout:      90 * time.Second,
	})
	handler := discordtransport.New(appID, svc)

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal(err)
	}
	dg.AddHandler(handler.OnInteractionCreate)
	if err := dg.Open(); err != nil {
		log.Fatal(err)
	}
	defer dg.Close()

	if err := handler.RegisterCommands(dg, guildID); err != nil {
		log.Printf("register commands: %v", err)
	}
	log.Println("Bot started")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
