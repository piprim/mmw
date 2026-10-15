package new

import (
	"testing"

	"github.com/piprim/goplt"
	"github.com/stretchr/testify/assert"
)

func TestOrderVars(t *testing.T) {
	names := func(m *goplt.Manifest) []string {
		out := make([]string, 0, len(m.Variables))
		for _, v := range m.Variables {
			out = append(out, v.Name)
		}

		return out
	}

	t.Run("name and entity come before the others", func(t *testing.T) {
		m := &goplt.Manifest{Variables: []goplt.Variable{{Name: "CreateTable"}, {Name: "Entity"}, {Name: "Name"}}}
		orderVars(m)
		assert.Equal(t, []string{"Name", "Entity", "CreateTable"}, names(m))
	})

	t.Run("unlisted variables keep their order", func(t *testing.T) {
		m := &goplt.Manifest{Variables: []goplt.Variable{{Name: "A"}, {Name: "Name"}, {Name: "B"}}}
		orderVars(m)
		assert.Equal(t, []string{"Name", "A", "B"}, names(m))
	})

	t.Run("database, create-table and grpc follow entity in that order", func(t *testing.T) {
		m := &goplt.Manifest{Variables: []goplt.Variable{
			{Name: "CreateTable"}, {Name: "Database"}, {Name: "Entity"}, {Name: "Grpc"}, {Name: "Name"},
		}}
		orderVars(m)
		assert.Equal(t, []string{"Name", "Entity", "Database", "CreateTable", "Grpc"}, names(m))
	})
}
