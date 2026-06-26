package httpapi

import (
	"xmine/litebans-api/internal/api"
	"xmine/litebans-api/internal/domain"
)

func toAPIModerator(m domain.Moderator) api.Moderator {
	return api.Moderator{Uuid: m.UUID, Name: m.Name, IsConsole: m.IsConsole}
}

func toAPIPlayer(p domain.Player) api.Player {
	return api.Player{Uuid: p.UUID, Name: p.Name, IsConsole: p.IsConsole, OfflineMode: p.OfflineMode}
}

func toAPIRemoved(r *domain.Removed) *api.Removed {
	if r == nil {
		return nil
	}
	return &api.Removed{
		By:                   toAPIModerator(r.By),
		At:                   r.At,
		Reason:               r.Reason,
		ExpiredAutomatically: r.ExpiredAutomatically,
	}
}

func toAPIPunishment(p domain.Punishment) api.Punishment {
	return api.Punishment{
		Id:           p.ID,
		Type:         api.PunishmentType(p.Type),
		PlayerUuid:   p.PlayerUUID,
		Reason:       p.Reason,
		Moderator:    toAPIModerator(p.Moderator),
		IssuedAt:     p.IssuedAt,
		ExpiresAt:    p.ExpiresAt,
		Permanent:    p.Permanent,
		Active:       p.Active,
		Expired:      p.Expired,
		IpBan:        p.IPBan,
		Silent:       p.Silent,
		ServerOrigin: p.ServerOrigin,
		ServerScope:  p.ServerScope,
		Removed:      toAPIRemoved(p.Removed),
		Acknowledged: p.Acknowledged,
	}
}

func toAPIPunishmentList(l domain.PunishmentList) api.PunishmentList {
	items := make([]api.Punishment, 0, len(l.Items))
	for _, item := range l.Items {
		items = append(items, toAPIPunishment(item))
	}
	return api.PunishmentList{
		Items:      items,
		Page:       int32(l.Page),
		PageSize:   int32(l.PageSize),
		TotalItems: l.TotalItems,
		TotalPages: int32(l.TotalPages),
	}
}

func toAPIStats(s domain.Stats) api.Stats {
	return api.Stats{Bans: s.Bans, Mutes: s.Mutes, Warnings: s.Warnings, Kicks: s.Kicks}
}
