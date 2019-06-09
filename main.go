package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
)

var (
	port              = os.Getenv("PORT")
	discordWebhookURL = os.Getenv("DISCORD_WEBHOOK_URL")
	msgPrefix = os.Getenv("MESSAGE_PREFIX")
	channelID = os.Getenv("SLACK_CHANNEL_ID")
)

var urlRe = regexp.MustCompile(`[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,6}\b([-a-zA-Z0-9@:%_\+.~#?&//=]*)`)

type slackTyped struct {
	Type string
}

type slackEventTyped struct {
	Event struct {
		Type string
	}
}

type slackChallengeEvent struct {
	Challenge string `json:"challenge"`
}

type slackMessageEvent struct {
	Event struct {
		Text string
		Channel string
	}
}

type discordWebhookRequest struct {
	Username string `json:"username"`
	Content  string `json:"content"`
}

func main() {
	if port == "" {
		port = "8080"
	}
	if discordWebhookURL == "" {
		fmt.Fprintln(os.Stderr, "No Discord webhook token is provided. Set DISCORD_WEBHOOK_TOKEN")
		os.Exit(1)
	}
	http.HandleFunc("/event", func(w http.ResponseWriter, r *http.Request) {
		var req slackTyped
		body, err := ioutil.ReadAll(r.Body)
		fmt.Println(string(body))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		if err := json.Unmarshal(body, &req); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		switch req.Type {
		case "url_verification":
			var challenge slackChallengeEvent
			if err := json.Unmarshal(body, &challenge); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}

			if err := json.NewEncoder(w).Encode(challenge); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}

		case "event_callback":
			var ev slackEventTyped
			if err := json.Unmarshal(body, &ev); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}

			switch ev.Event.Type {
			case "message":
				var msg slackMessageEvent
				if err := json.Unmarshal(body, &msg); err != nil {
					fmt.Fprintln(os.Stderr, err)
					return
				}

				if msg.Event.Channel != channelID {
					return
				}

				for _, url := range urlRe.FindAllString(msg.Event.Text, -1) {
					wreq := &discordWebhookRequest{
						Username: "Music Commander",
						Content:  fmt.Sprintf("%s%s", msgPrefix, url),
					}
					r, w := io.Pipe()
					go func() {
						if err := json.NewEncoder(w).Encode(wreq); err != nil {
							fmt.Fprintln(os.Stderr, err)
						}
						w.Close()
					}()
					resp, err := http.Post(discordWebhookURL, "application/json", r)
					if err != nil {
						fmt.Fprintln(os.Stderr, err)
						return
					}
					switch resp.StatusCode {
					case http.StatusOK:
						fmt.Printf("%s; Response: ", resp.Status)
						io.Copy(os.Stdout, resp.Body)
						fmt.Println()
					default:
						fmt.Fprintf(os.Stderr, "%s; Response: ", resp.Status)
						io.Copy(os.Stderr, resp.Body)
						fmt.Fprintln(os.Stderr)
					}
				}
			default:
				fmt.Printf("Ignored: %s\n", string(body))
			}
		default:
			fmt.Printf("Ignored: %s\n", string(body))
		}
	})
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
