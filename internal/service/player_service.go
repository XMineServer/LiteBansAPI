package service

import (
	"context"
	"database/sql"
	"strings"
	"xmine/litebans-api/internal/domain"
	"xmine/litebans-api/internal/repository"
)

type PlayerService struct {
	historyRepo    *repository.HistoryRepository
	consoleAliases map[string]struct{}
}

func NewPlayerService(historyRepo *repository.HistoryRepository, consoleAliases []string) *PlayerService {
	set := make(map[string]struct{}, len(consoleAliases))
	for _, alias := range consoleAliases {
		set[alias] = struct{}{}
	}
	return &PlayerService{historyRepo: historyRepo, consoleAliases: set}
}

func (s *PlayerService) isAliasName(name string) bool {
	_, ok := s.consoleAliases[name]
	return ok
}

// IsOfflineMode reports whether a uuid was generated offline-mode (version 3, name-derived)
// rather than issued by Mojang's online auth (version 4). See TOR 7.2.
func IsOfflineMode(uuid string) bool {
	clean := strings.ReplaceAll(uuid, "-", "")
	if len(clean) < 13 {
		return false
	}
	return clean[12] == '3'
}

// ResolveModerator resolves the moderator/remover identity for a punishment row's banned_by_*
// or removed_by_* columns, per TOR 3.4/6.4: console aliases skip NameHistory lookup entirely,
// otherwise the current name is resolved via history with the stored snapshot as fallback.
func (s *PlayerService) ResolveModerator(ctx context.Context, uuid string, snapshotName sql.NullString) (domain.Moderator, error) {
	if uuid == "" || (snapshotName.Valid && s.isAliasName(snapshotName.String)) {
		var namePtr *string
		if snapshotName.Valid {
			n := snapshotName.String
			namePtr = &n
		}
		return domain.Moderator{UUID: nil, Name: namePtr, IsConsole: true}, nil
	}

	resolved, err := s.historyRepo.LatestNameByUUID(ctx, uuid)
	if err != nil {
		return domain.Moderator{}, err
	}
	name := resolved
	if name == "" && snapshotName.Valid {
		name = snapshotName.String
	}
	var namePtr *string
	if name != "" {
		namePtr = &name
	}
	return domain.Moderator{UUID: &uuid, Name: namePtr, IsConsole: false}, nil
}

// ResolvePlayerByUUID resolves a Player identity for the /players/lookup endpoint given a uuid.
func (s *PlayerService) ResolvePlayerByUUID(ctx context.Context, uuid string) (domain.Player, bool, error) {
	name, err := s.historyRepo.LatestNameByUUID(ctx, uuid)
	if err != nil {
		return domain.Player{}, false, err
	}
	if name == "" {
		return domain.Player{}, false, nil
	}
	uuidVal := uuid
	nameVal := name
	return domain.Player{
		UUID:        &uuidVal,
		Name:        &nameVal,
		IsConsole:   false,
		OfflineMode: IsOfflineMode(uuid),
	}, true, nil
}

// ResolvePlayerByName resolves a Player identity for the /players/lookup endpoint given a name.
// If the name matches a configured console alias, no NameHistory lookup is performed.
func (s *PlayerService) ResolvePlayerByName(ctx context.Context, name string) (domain.Player, bool, error) {
	if s.isAliasName(name) {
		nameVal := name
		return domain.Player{UUID: nil, Name: &nameVal, IsConsole: true, OfflineMode: false}, true, nil
	}

	uuid, err := s.historyRepo.LatestUUIDByName(ctx, name)
	if err != nil {
		return domain.Player{}, false, err
	}
	if uuid == "" {
		return domain.Player{}, false, nil
	}
	uuidVal := uuid
	nameVal := name
	return domain.Player{
		UUID:        &uuidVal,
		Name:        &nameVal,
		IsConsole:   false,
		OfflineMode: IsOfflineMode(uuid),
	}, true, nil
}
