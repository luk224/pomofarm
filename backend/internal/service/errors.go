package service

import "errors"

// Domain errors; the api package maps them to HTTP statuses.
var (
	ErrNoPlayer          = errors.New("no_player")
	ErrNotFound          = errors.New("not_found")
	ErrInvalid           = errors.New("invalid_request")
	ErrSeedLocked        = errors.New("seed_locked")
	ErrPomodoroActive    = errors.New("pomodoro_active")
	ErrNoActivePomodoro  = errors.New("no_active_pomodoro")
	ErrPlotBusy          = errors.New("plot_busy")
	ErrNotMature         = errors.New("not_mature")
	ErrAlreadyHarvested  = errors.New("already_harvested")
	ErrWrongState        = errors.New("wrong_state") // e.g. pausing a paused Pomodoro
	ErrConflict          = errors.New("version_conflict")
	ErrAlreadyUnlocked   = errors.New("already_unlocked")
	ErrHarvestFirst      = errors.New("harvest_first")
	ErrSiloEmpty         = errors.New("silo_empty")
	ErrMaxedOut          = errors.New("maxed_out")
	ErrNoRest            = errors.New("no_rest")
	ErrNeedsConfirmation = errors.New("needs_confirmation")
	ErrInsufficientFocus = errors.New("insufficient_focus")
	ErrInsufficientCoins = errors.New("insufficient_coins")
	ErrAlreadyOwned      = errors.New("already_owned")
	ErrCellTaken         = errors.New("cell_taken")
	ErrAnimalLocked      = errors.New("animal_locked")
	ErrBadCell           = errors.New("bad_cell")
)
