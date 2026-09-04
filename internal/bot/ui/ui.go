// Package ui builds the bot's Discord messages. Everything is rendered with
// Components V2 containers rather than embeds.
package ui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/disgoorg/disgo/discord"
)

// Accent colours, one per kind of response.
const (
	ColorContent = 0x7c5cff
	ColorSuccess = 0x57f287
	ColorError   = 0xed4245
	ColorWrapped = 0xeb459e
)

// Custom IDs routed by the interaction handler.
const (
	IDToggle     = "/player/toggle"
	IDSkip       = "/player/skip"
	IDStop       = "/player/stop"
	IDNowPlaying = "/player/nowplaying"
	IDQueue      = "/player/queue"
	IDSearchPlay = "/search/play"
	IDAlbumPlay  = "/album/play/"

	IDCaptionsStop = "/captions/stop"
)

// Components V2 allows 4000 characters of text per message across every text
// display. Lists get most of it; the rest covers headers and footers, and fit
// enforces the ceiling for anything that still overruns.
const (
	messageBudget = 4000
	listBudget    = 3600
	barWidth      = 18
)

var markdown = strings.NewReplacer(
	`\`, `\\`, "*", `\*`, "_", `\_`, "~", `\~`, "`", "\\`", "|", `\|`, ">", `\>`, "#", `\#`,
)

// Reply builds a Components V2 edit for an already-deferred interaction.
func Reply(color int, components ...discord.ContainerSubComponent) discord.MessageUpdate {
	flags := discord.MessageFlagIsComponentsV2
	layout := []discord.LayoutComponent{discord.NewContainer(fit(components)...).WithAccentColor(color)}
	return discord.MessageUpdate{Components: &layout, Flags: &flags}
}

// fit trims text to the per-message budget, keeping interactive rows intact.
func fit(components []discord.ContainerSubComponent) []discord.ContainerSubComponent {
	remaining := messageBudget
	kept := make([]discord.ContainerSubComponent, 0, len(components))
	for _, component := range components {
		switch typed := component.(type) {
		case discord.TextDisplayComponent:
			text, used := clamp(typed.Content, remaining)
			if text == "" {
				continue
			}
			remaining -= used
			kept = append(kept, typed.WithContent(text))
		case discord.SectionComponent:
			texts := make([]discord.SectionSubComponent, 0, len(typed.Components))
			for _, sub := range typed.Components {
				display, ok := sub.(discord.TextDisplayComponent)
				if !ok {
					texts = append(texts, sub)
					continue
				}
				text, used := clamp(display.Content, remaining)
				if text == "" {
					continue
				}
				remaining -= used
				texts = append(texts, display.WithContent(text))
			}
			if len(texts) == 0 {
				continue
			}
			kept = append(kept, typed.WithComponents(texts...))
		default:
			kept = append(kept, component)
		}
	}
	return kept
}

func clamp(text string, remaining int) (string, int) {
	length := utf8.RuneCountInString(text)
	switch {
	case remaining <= 0:
		return "", 0
	case length <= remaining:
		return text, length
	default:
		return Truncate(text, remaining), remaining
	}
}

// Fail renders a user-facing problem. Its message is the whole response.
func Fail(message string) discord.MessageUpdate {
	return Reply(ColorError, discord.NewTextDisplay("### "+message))
}

// Note renders a short confirmation.
func Note(color int, format string, a ...any) discord.MessageUpdate {
	return Reply(color, discord.NewTextDisplay("### "+fmt.Sprintf(format, a...)))
}

// Attach replaces the message's uploads, which is how cover art reaches Discord
// on servers that are not publicly reachable.
func Attach(update discord.MessageUpdate, files ...*discord.File) discord.MessageUpdate {
	update.Files = files
	update.Attachments = &[]discord.AttachmentUpdate{}
	return update
}

// Text is a markdown paragraph. Callers format their own content.
func Text(format string, a ...any) discord.TextDisplayComponent {
	return discord.NewTextDisplay(fmt.Sprintf(format, a...))
}

// Escape neutralises markdown in library text so titles render literally.
func Escape(s string) string { return markdown.Replace(s) }

// Truncate shortens text to max runes, ending in an ellipsis.
func Truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return strings.TrimRight(string(runes[:max-1]), " ") + "…"
}

// Join renders list lines within Discord's text budget and says what was cut.
func Join(lines []string) string {
	var out strings.Builder
	used := 0
	for i, line := range lines {
		used += utf8.RuneCountInString(line) + 1
		if used > listBudget {
			fmt.Fprintf(&out, "\n-# and %d more", len(lines)-i)
			break
		}
		if i > 0 {
			out.WriteByte('\n')
		}
		out.WriteString(line)
	}
	return out.String()
}

// Duration formats a track length in seconds.
func Duration(seconds int) string {
	if seconds <= 0 {
		return "--:--"
	}
	return Clock(time.Duration(seconds) * time.Second)
}

// Clock formats a playback position as m:ss, or h:mm:ss past an hour.
func Clock(d time.Duration) string {
	total := int(d.Seconds())
	hours, minutes, seconds := total/3600, total/60%60, total%60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// Progress renders the elapsed position as a text bar.
func Progress(elapsed time.Duration, totalSeconds int) string {
	total := time.Duration(totalSeconds) * time.Second
	if total <= 0 {
		return fmt.Sprintf("`%s`", Clock(elapsed))
	}
	filled := int(float64(barWidth) * min(elapsed.Seconds()/total.Seconds(), 1))
	bar := strings.Repeat("▰", filled) + strings.Repeat("▱", barWidth-filled)
	return fmt.Sprintf("`%s` %s `%s`", Clock(elapsed), bar, Clock(total))
}

// Minutes renders a listening total in hours and minutes.
func Minutes(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%d min", minutes)
	}
	return fmt.Sprintf("%dh %dm", minutes/60, minutes%60)
}

// Controls is the playback button row shared by the now playing and queue views.
func Controls(paused bool) discord.ActionRowComponent {
	toggle := discord.NewSecondaryButton("Pause", IDToggle)
	if paused {
		toggle = discord.NewSuccessButton("Resume", IDToggle)
	}
	return discord.NewActionRow(
		toggle,
		discord.NewSecondaryButton("Skip", IDSkip),
		discord.NewSecondaryButton("Queue", IDQueue),
		discord.NewSecondaryButton("Now playing", IDNowPlaying),
		discord.NewDangerButton("Stop", IDStop),
	)
}

// Create turns a rendered reply into a standalone message. Captions live in a
// plain channel message, since interaction tokens expire after 15 minutes and a
// listening session outlasts that.
func Create(update discord.MessageUpdate) discord.MessageCreate {
	create := discord.MessageCreate{}
	if update.Components != nil {
		create.Components = *update.Components
	}
	if update.Flags != nil {
		create.Flags = *update.Flags
	}
	return create
}
