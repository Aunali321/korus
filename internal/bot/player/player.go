// Package player runs one voice playback session per guild.
package player

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"

	"github.com/Aunali321/korus/internal/bot/korus"
)

// ErrBusy reports that the guild already has a session in another voice channel.
var ErrBusy = errors.New("player: session is in another voice channel")

// Track is one queued song.
//
// Listener is the account the play is recorded to: whoever queued the track,
// and only when they are linked to the same Korus server the session streams
// from. A guest queueing out of the host's library therefore never shows up in
// the host's listening history.
type Track struct {
	Song      korus.Song
	Listener  *korus.Client
	Requester string
}

// Snapshot is a consistent view of a session for rendering.
type Snapshot struct {
	Current Track
	Elapsed time.Duration
	Paused  bool
	Playing bool
	Queue   []Track
}

type Player struct {
	guildID   snowflake.ID
	channelID snowflake.ID
	host      snowflake.ID
	source    *korus.Client
	conn      voice.Conn
	ffmpeg    string
	log       *slog.Logger
	release   func()

	mu      sync.Mutex
	queue   []Track
	current *Track
	stream  *stream

	skip       chan struct{}
	stop       chan struct{}
	signalStop func()
}

func (p *Player) GuildID() snowflake.ID { return p.guildID }
func (p *Player) Host() snowflake.ID    { return p.host }

func (p *Player) ChannelID() snowflake.ID {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.channelID
}

// SetChannel records where the session ended up after Discord moved the bot.
func (p *Player) SetChannel(channelID snowflake.ID) {
	p.mu.Lock()
	p.channelID = channelID
	p.mu.Unlock()
}

// Source is the host's library: every track in the session streams from it.
func (p *Player) Source() *korus.Client { return p.source }

// Enqueue appends tracks and returns the queue position of the first one,
// counting the playing track as position zero.
func (p *Player) Enqueue(tracks ...Track) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	position := len(p.queue)
	if p.current != nil {
		position++
	}
	p.queue = append(p.queue, tracks...)
	return position
}

func (p *Player) Snapshot() Snapshot {
	p.mu.Lock()
	defer p.mu.Unlock()

	snap := Snapshot{Queue: slices.Clone(p.queue)}
	if p.current != nil {
		snap.Current = *p.current
		snap.Playing = true
	}
	if p.stream != nil {
		snap.Elapsed = p.stream.elapsed()
		snap.Paused = p.stream.isPaused()
	}
	return snap
}

// Pause holds playback. It reports false when playback was already paused.
func (p *Player) Pause() bool { return p.setPaused(true) }

// Resume releases the pause gate. It reports false when already playing.
func (p *Player) Resume() bool { return p.setPaused(false) }

func (p *Player) setPaused(paused bool) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stream == nil || p.stream.isPaused() == paused {
		return false
	}
	p.stream.setPaused(paused)
	return true
}

// SkipResult describes a skip at the moment it was requested.
type SkipResult struct {
	Skipped  Track
	Upcoming []Track
}

// Skip ends the current track. The result is read under the queue lock, so it
// describes what happens next without racing the playback loop.
func (p *Player) Skip() (SkipResult, bool) {
	p.mu.Lock()
	if p.current == nil {
		p.mu.Unlock()
		return SkipResult{}, false
	}
	result := SkipResult{Skipped: *p.current, Upcoming: slices.Clone(p.queue)}
	p.mu.Unlock()

	select {
	case p.skip <- struct{}{}:
	default:
	}
	return result, true
}

// Stop clears the queue and ends the session. The run loop leaves the channel.
func (p *Player) Stop() {
	p.mu.Lock()
	p.queue = nil
	p.mu.Unlock()
	p.signalStop()
}

func (p *Player) run() {
	defer p.leave()
	for {
		track, ok := p.dequeue()
		if !ok {
			return
		}
		if !p.play(track) {
			return
		}
	}
}

func (p *Player) dequeue() (Track, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.queue) == 0 {
		p.current = nil
		return Track{}, false
	}
	track := p.queue[0]
	p.queue = p.queue[1:]
	p.current = &track
	return track, true
}

// play streams one track and reports whether the session should continue.
func (p *Player) play(track Track) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	bearer, err := p.source.Bearer(ctx)
	cancel()
	if err != nil {
		p.log.Error("cannot authenticate stream", "song", track.Song.ID, "err", err)
		return !p.stopped()
	}

	current, err := newStream(p.ffmpeg, p.source.DownloadURL(track.Song.ID), bearer)
	if err != nil {
		p.log.Error("cannot start ffmpeg", "song", track.Song.ID, "err", err)
		return !p.stopped()
	}

	p.mu.Lock()
	p.stream = current
	p.mu.Unlock()

	select {
	case <-p.skip:
	default:
	}

	p.conn.SetOpusFrameProvider(current)

	stopped := false
	select {
	case <-current.done:
		if stderr := current.err(); stderr != "" {
			p.log.Error("ffmpeg reported an error", "song", track.Song.ID, "stderr", stderr)
		}
	case <-p.skip:
		current.Close()
	case <-p.stop:
		current.Close()
		stopped = true
	}

	p.mu.Lock()
	p.stream = nil
	p.mu.Unlock()

	p.record(track, current.elapsed())
	return !stopped
}

func (p *Player) stopped() bool {
	select {
	case <-p.stop:
		return true
	default:
		return false
	}
}

func (p *Player) record(track Track, elapsed time.Duration) {
	seconds := int(elapsed.Seconds())
	if track.Listener == nil || seconds == 0 {
		return
	}
	completion := 0.0
	if total := track.Song.Seconds(); total > 0 {
		completion = min(float64(seconds)/float64(total), 1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	listen := korus.Listen{
		SongID:           track.Song.ID,
		DurationListened: seconds,
		CompletionRate:   completion,
		Source:           "discord",
	}
	if err := track.Listener.RecordListen(ctx, listen); err != nil {
		p.log.Error("cannot record listen", "song", track.Song.ID, "err", err)
	}
}

func (p *Player) leave() {
	p.mu.Lock()
	p.current, p.queue = nil, nil
	p.mu.Unlock()

	p.conn.SetOpusFrameProvider(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p.conn.Close(ctx)
	p.release()
}

// Manager owns one Player per guild.
type Manager struct {
	client *bot.Client
	ffmpeg string
	log    *slog.Logger

	mu      sync.Mutex
	players map[snowflake.ID]*Player
}

func NewManager(client *bot.Client, ffmpeg string, log *slog.Logger) *Manager {
	return &Manager{
		client:  client,
		ffmpeg:  ffmpeg,
		log:     log,
		players: map[snowflake.ID]*Player{},
	}
}

func (m *Manager) Get(guildID snowflake.ID) *Player {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.players[guildID]
}

// Join returns the guild's session, creating one hosted by host when there is
// none. An existing session only moves channel for its own host.
func (m *Manager) Join(ctx context.Context, guildID, channelID, host snowflake.ID, source *korus.Client) (*Player, error) {
	m.mu.Lock()
	if existing, ok := m.players[guildID]; ok {
		m.mu.Unlock()
		if existing.ChannelID() == channelID {
			return existing, nil
		}
		if existing.Host() != host {
			return nil, ErrBusy
		}
		return existing, m.move(ctx, existing, channelID)
	}

	player := &Player{
		guildID:   guildID,
		channelID: channelID,
		host:      host,
		source:    source,
		conn:      m.client.VoiceManager.CreateConn(guildID),
		ffmpeg:    m.ffmpeg,
		log:       m.log.With("guild", guildID),
		release:   func() { m.remove(guildID) },
		skip:      make(chan struct{}, 1),
		stop:      make(chan struct{}),
	}
	player.signalStop = sync.OnceFunc(func() { close(player.stop) })
	m.players[guildID] = player
	m.mu.Unlock()

	if err := player.conn.Open(ctx, channelID, false, false); err != nil {
		m.remove(guildID)
		m.client.VoiceManager.RemoveConn(guildID)
		return nil, err
	}
	go player.run()
	return player, nil
}

// move re-targets a live session. Discord keeps the voice session alive across
// a channel change within a guild, so this is a plain state update with nothing
// to wait for, unlike the handshake Conn.Open performs.
func (m *Manager) move(ctx context.Context, target *Player, channelID snowflake.ID) error {
	data := gateway.MessageDataVoiceStateUpdate{GuildID: target.GuildID(), ChannelID: &channelID}
	if err := m.client.Gateway.Send(ctx, gateway.OpcodeVoiceStateUpdate, data); err != nil {
		return err
	}
	target.SetChannel(channelID)
	return nil
}

func (m *Manager) remove(guildID snowflake.ID) {
	m.mu.Lock()
	delete(m.players, guildID)
	m.mu.Unlock()
}

// StopAll ends every session, used on shutdown.
func (m *Manager) StopAll() {
	m.mu.Lock()
	players := make([]*Player, 0, len(m.players))
	for _, player := range m.players {
		players = append(players, player)
	}
	m.mu.Unlock()
	for _, player := range players {
		player.Stop()
	}
}
