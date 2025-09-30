package user

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	MinPermission = 0
	MaxPermission = 255
)

type Permission = uint8

type Group struct {
	name       string
	permission Permission
}

type groupJson struct {
	Name       string `json:"name"`
	Permission int    `json:"permission"`
}

func newPermission(p int) Permission {
	return Permission(max(MinPermission, min(MaxPermission, p)))
}

func NewGroup(name string, permission int) Group {
	return Group{ name: name, permission: newPermission(permission), }
}

func EmptyGroup() Group {
	return Group { name: "anon", permission: MinPermission }
}

func AdminGroup() Group {
	return Group { name: "admin", permission: MaxPermission }
}

func (g Group) Name() string {
	return g.name
}

func (g Group) Permission() int {
	return int(g.permission)
}

func (g Group) WithName(name string) Group {
	return Group{ name: name, permission: g.permission }
}

func (g Group) WithPermission(permission int) Group {
	return Group{ name: g.name, permission: newPermission(permission), }
}

func (g Group) String() string {
	return fmt.Sprintf("<group %s (%d)>", g.name, g.permission)
}

func (g Group) MarshalJSON() ([]byte, error) {
	return json.Marshal(groupJson{
		Name:         g.name,
		Permission:   int(g.permission),
	})
}

func (g *Group) UnmarshalJSON(b []byte) error {
	var data groupJson
	err := json.Unmarshal(b, &data)
	if err != nil {
		return err
	}
	g.name = data.Name
	g.permission = newPermission(data.Permission)
	return nil
}

func CompareGroups(g Group, h Group) int {
	res := g.Permission() - h.Permission()
	if res == 0 {
		res = strings.Compare(g.Name(), h.Name())
	}
	return res
}
