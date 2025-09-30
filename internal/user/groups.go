package user

import (
	"fmt"
	"iter"
	"log/slog"
	"slices"
	"sort"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
)

var (
	groupsByName map[string]Group
	groups []Group
)

type UsersInGroup struct {
	Group string
	Users []string
}

func addFixedGroup(g Group) {
	p, exists := cfg.CustomGroups[g.Name()]
	if exists && p != g.Permission() {
		slog.Warn(fmt.Sprintf(
			"The permission level of the fixed group '%s' is configured to %d; resetting to %d",
			g.Name(), p, g.Permission(),
		))
	}
	cfg.CustomGroups[g.Name()] = g.Permission()
}

func initGroups() error {
	var gs []Group
	if cfg.CustomGroups == nil {
		gs = []Group{
			EmptyGroup(),
			NewGroup("reader", 0),
			NewGroup("editor", 1),
			NewGroup("trusted", 2),
			NewGroup("moderator", 3),
			AdminGroup(),
		}
	} else {
		addFixedGroup(EmptyGroup())
		addFixedGroup(AdminGroup())
		gs = make([]Group, len(cfg.CustomGroups))
		i := 0
		for k, v := range cfg.CustomGroups {
			gs[i] = NewGroup(k, v)
			i++
		}
	}
	setGroups(gs)
	slog.Info("Indexed groups", "n", len(groups))
	return nil
}

func setGroups(gs []Group) {
	slices.SortFunc(gs, CompareGroups)
	gsByName := make(map[string]Group)
	for _, g := range gs {
		gsByName[g.Name()] = g
	}
	groups = gs
	groupsByName = gsByName
}

func GroupByName(name string) (Group, error) {
	g, ok := groupsByName[name]
	if !ok {
		return EmptyGroup(), fmt.Errorf("group '%s' does not exist", name)
	}
	return g, nil
}

func YieldGroups() iter.Seq[Group] {
	return func(yield func(Group) bool) {
		for _, g := range groups {
			if !yield(g) {
				return
			}
		}
	}
}

func Groups() []Group {
	res := make([]Group, len(groups))
	copy(res, groups)
	return res
}

func UsersInGroups() []UsersInGroup {
	res := make([]UsersInGroup, len(groups))
	index := make(map[string]int)
	for i, g := range groups {
		name := g.Name()
		res[i].Group = name
		index[name] = i
	}
	for u := range YieldUsers() {
		if i, ok := index[u.GroupName()]; ok {
			res[i].Users = append(res[i].Users, u.Name())
		}
	}
	for _, r := range res {
		sort.Strings(r.Users)
	}
	return res
}
