package api

import (
	"strconv"
	"testing"

	"github.com/luk224/pomofarm/backend/internal/game"
)

func buyDecor(e *env, code int, kind string, x, y int) []byte {
	e.t.Helper()
	return e.expect(code, "POST", "/api/decor", map[string]any{"kind": kind, "x": x, "y": y})
}

func TestDecorCatalogAndPrices(t *testing.T) {
	e := newEnv(t, 1)
	d := e.state().Decor
	if len(d.Catalog) != 3 || d.Catalog[0].Kind != "path" || d.Catalog[0].Cost != 15 || d.Catalog[1].Cost != 25 || d.Catalog[2].Cost != 80 {
		t.Fatalf("catalog = %+v", d.Catalog)
	}
	if d.Hat.Cost != 600 || d.Hat.Available || d.Hat.Owned || len(d.Items) != 0 {
		t.Fatalf("hat/items = %+v", d)
	}
	giveCoins(t, e, 1000)
	for i, c := range []struct {
		kind string
		cost int64
	}{{"path", 15}, {"fence", 25}, {"lantern", 80}} {
		before := coinsOf(e)
		buyDecor(e, 201, c.kind, i, -2) // distinct cells on the top row
		if got := before - coinsOf(e); got != c.cost*1000 {
			t.Fatalf("%s cost %d milli, want %d", c.kind, got, c.cost*1000)
		}
	}
	if n := len(e.state().Decor.Items); n != 3 {
		t.Fatalf("items = %d", n)
	}
}

func TestDecorRejectsBadCells(t *testing.T) {
	e := newEnv(t, 1)
	giveCoins(t, e, 1000)
	for _, c := range [][2]int{{0, 0}, {3, 3}, {1, 2}, {-3, 0}, {6, 0}, {0, 6}, {-1, 4}, {0, 4}} {
		buyDecor(e, 400, "path", c[0], c[1])
	}
	buyDecor(e, 400, "statue", -2, -2)
	if coinsOf(e) != 1000*1000 || len(e.state().Decor.Items) != 0 {
		t.Fatal("a rejected purchase must cost nothing and place nothing")
	}
	buyDecor(e, 201, "path", -2, 0) // edge of the allowed area
	buyDecor(e, 201, "path", 5, 5)
}

func TestDecorOneCellOnePieceAndNoCharge(t *testing.T) {
	e := newEnv(t, 1)
	giveCoins(t, e, 1000)
	buyDecor(e, 201, "lantern", 4, 1)
	before := coinsOf(e)
	buyDecor(e, 409, "path", 4, 1)
	if coinsOf(e) != before {
		t.Fatal("a refused placement must not charge")
	}
}

func TestDecorNeedsCoins(t *testing.T) {
	e := newEnv(t, 1)
	giveCoins(t, e, 14)
	buyDecor(e, 409, "path", 4, 1)
	giveCoins(t, e, 15)
	buyDecor(e, 201, "path", 4, 1)
	if coinsOf(e) != 0 {
		t.Fatalf("coins = %d", coinsOf(e))
	}
}

func TestDecorMoveAndRemoveAreFree(t *testing.T) {
	e := newEnv(t, 1)
	giveCoins(t, e, 500)
	buyDecor(e, 201, "fence", 4, 1)
	buyDecor(e, 201, "fence", 4, 2)
	id := e.state().Decor.Items[0].ID
	before := coinsOf(e)
	e.expect(200, "POST", "/api/decor/"+strconv.FormatInt(id, 10)+"/move", map[string]any{"x": 5, "y": 1})
	e.expect(409, "POST", "/api/decor/"+strconv.FormatInt(id, 10)+"/move", map[string]any{"x": 5, "y": 1}) // same cell
	e.expect(409, "POST", "/api/decor/"+strconv.FormatInt(id, 10)+"/move", map[string]any{"x": 4, "y": 2}) // taken
	e.expect(400, "POST", "/api/decor/"+strconv.FormatInt(id, 10)+"/move", map[string]any{"x": 1, "y": 1}) // the field
	e.expect(404, "POST", "/api/decor/9999/move", map[string]any{"x": 5, "y": 5})
	it := e.state().Decor.Items[0]
	if it.X != 5 || it.Y != 1 || coinsOf(e) != before {
		t.Fatalf("after moves: %+v, coins %d (was %d)", it, coinsOf(e), before)
	}
	e.expect(200, "DELETE", "/api/decor/"+strconv.FormatInt(id, 10), nil)
	e.expect(404, "DELETE", "/api/decor/"+strconv.FormatInt(id, 10), nil)
	if len(e.state().Decor.Items) != 1 || coinsOf(e) != before {
		t.Fatal("removing leaves the other piece and does not touch coins")
	}
	buyDecor(e, 201, "path", 5, 1) // the cell is free again
}

func TestDecorNeverTouchesProductionOrBonuses(t *testing.T) {
	e := newEnv(t, 1)
	giveCoins(t, e, 5000)
	rate := e.state().Silo.RateMilliPerHour
	for x := -2; x <= 5; x++ {
		buyDecor(e, 201, "lantern", x, -2)
	}
	st := e.state()
	if st.Silo.RateMilliPerHour != rate || len(st.Plots) != 1 || st.Player.Season != 1 {
		t.Fatal("decoration must not change production, plots or prestige")
	}
}

func TestHatNeedsTheDog(t *testing.T) {
	e := newEnv(t, 1)
	giveCoins(t, e, 100000)
	unlockAnimals(t, e)
	e.expect(403, "POST", "/api/decor", map[string]any{"kind": "hat"})
	e.expect(201, "POST", "/api/structures", map[string]any{"kind": "dog"})
	if h := e.state().Decor.Hat; !h.Available || h.Owned {
		t.Fatalf("hat = %+v", h)
	}
	before := coinsOf(e)
	e.expect(201, "POST", "/api/decor", map[string]any{"kind": "hat"})
	if before-coinsOf(e) != int64(game.HatCost)*1000 {
		t.Fatal("hat must cost 600 🪙")
	}
	e.expect(409, "POST", "/api/decor", map[string]any{"kind": "hat"})
	if h := e.state().Decor.Hat; h.Available || !h.Owned {
		t.Fatalf("hat = %+v", h)
	}
}

func TestHatNeedsCoins(t *testing.T) {
	e := newEnv(t, 1)
	giveCoins(t, e, 30000)
	unlockAnimals(t, e)
	e.expect(201, "POST", "/api/structures", map[string]any{"kind": "dog"})
	e.expect(409, "POST", "/api/decor", map[string]any{"kind": "hat"})
}
