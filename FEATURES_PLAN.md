# Feature Implementation Plan

## Game Mechanics Simulation
- Heat Card Tracker — Track each player's heat cards in engine/discard/hand
- Gear Shift & Stress Log — Log each player's gear choice and stress per lap
- Upgrade Cards Management — Track which upgrades each player has bought in a championship
- Legend/Sponsorship Abilities — Track legendary driver powers and sponsorship perks
- Deck Builder — Let players build custom upgrade decks before a season starts

## Multi-User Support
- Player Self-Service Mode — Players report their own gear/position from their phone
- Spectator Mode — Read-only view for non-players
- Shared Race Control — Multiple admins can control the race

## Race Enhancements
- Weather System — Dynamic wet/dry conditions that affect track grip
- Turbo Boost Log — Track when and how many times each player used turbo
- Corner/Sector Tracking — Break track into sectors instead of just % position
- Lap-by-Lap Replay — Timeline showing position changes each lap
- AI Difficulty Presets — Aggressive, balanced, conservative AI behavior profiles

## Implementation Phases

### Phase 1: Database Schema + Models
- New tables: heat_cards, gear_shifts, upgrades, player_sessions, weather, turbo_logs, lap_history
- New models structs in models/models.go
- Schema migration via db/db.go

### Phase 2: Backend Handlers
- Game mechanics API endpoints (heat cards CRUD, gear log, upgrades)
- Multi-user endpoints (player login, session management, self-service)
- Race enhancement endpoints (weather, turbo, lap history)

### Phase 3: WebSocket Enhancements
- Separate broadcast channels for game mechanics data
- Player-specific WebSocket channels for self-service
- Lap-by-lap replay broadcasting

### Phase 4: Frontend Pages
- Player dashboard (for self-service from phone)
- Race control enhancements (weather panel, turbo tracker, sector view)
- Upgrade/Deck builder UI
- Lap replay viewer
