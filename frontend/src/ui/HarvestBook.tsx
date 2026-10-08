import { useEffect, useRef, useState } from 'react'
import { api, ApiError, bookExportUrl } from '../api/client'
import type { BookMonth } from '../api/types'
import {
  dayLabel, formatFocus, harvestOf, HARVEST_UNITS, isHarvestUnit, monthLabel, UNIT_NAMES, WEEKDAY_NAMES, type HarvestUnit,
} from '../store/book'
import { HarvestIcon } from './HarvestIcons'
import { messageFor } from './messages'

const UNIT_KEY = 'pomofarm.book.unit'

function loadUnit(): HarvestUnit {
  try {
    const v = localStorage.getItem(UNIT_KEY)
    return isHarvestUnit(v) ? v : 'bale'
  } catch {
    return 'bale'
  }
}

function BookIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" focusable="false">
      <path d="M4 3.5h9.5a2 2 0 0 1 2 2v11H6a2 2 0 0 1-2-2zM6.5 6.5h6M6.5 9h6" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

/** The icons that stand for the month's hours of focus. */
function Harvest({ seconds, unit }: { seconds: number; unit: HarvestUnit }) {
  const h = harvestOf(seconds)
  const names = UNIT_NAMES[unit]
  const count = h.full + (h.part > 0 ? 1 : 0)
  return (
    <div className="harvest" data-testid="harvest">
      <div className="harvest__icons" role="img"
        aria-label={`${formatFocus(seconds)} de concentración: ${count === 0 ? 'ninguna' : `${count} ${count === 1 ? names.one : names.many}`}${h.hoursPerIcon > 1 ? `, cada una son ${h.hoursPerIcon} horas` : ''}`}>
        {Array.from({ length: h.full }, (_, i) => <HarvestIcon key={i} unit={unit} />)}
        {h.part > 0 && <HarvestIcon unit={unit} fill={h.part} />}
      </div>
      <p className="hint" data-testid="harvest-legend">
        {seconds === 0 ? 'Aquí aparecerá tu cosecha.' : `Cada ${names.one} son ${h.hoursPerIcon === 1 ? '1 hora' : `${h.hoursPerIcon} horas`} de concentración.`}
      </p>
    </div>
  )
}

/** Spanish weekday initials: L M X J V S D (X is Wednesday, so Martes and Miércoles differ). */
const WEEKDAY_LETTERS = ['L', 'M', 'X', 'J', 'V', 'S', 'D']

function Weekdays({ book }: { book: BookMonth }) {
  const max = Math.max(1, ...book.weekdays.map((w) => w.seconds))
  return (
    <div className="bars" role="group" aria-label="Concentración por día de la semana">
      {book.weekdays.map((w) => (
        <div key={w.weekday} className="bars__col" title={`${WEEKDAY_NAMES[w.weekday]}: ${formatFocus(w.seconds)}`}>
          <span className="bars__bar"><span style={{ height: `${(w.seconds / max) * 100}%` }} /></span>
          <span className="bars__label" aria-hidden="true">{WEEKDAY_LETTERS[w.weekday]}</span>
        </div>
      ))}
    </div>
  )
}

function Tables({ book }: { book: BookMonth }) {
  return (
    <div className="book__tables" data-testid="book-tables">
      <table className="table">
        <caption>Por etiqueta</caption>
        <thead><tr><th scope="col">Etiqueta</th><th scope="col">Pomodoros</th><th scope="col">Tiempo</th></tr></thead>
        <tbody>
          {book.tags.map((t) => (
            <tr key={t.name}><th scope="row">{t.name || 'Sin etiqueta'}</th><td>{t.pomodoros}</td><td>{formatFocus(t.seconds)}</td></tr>
          ))}
          {book.tags.length === 0 && <tr><td colSpan={3}>Sin datos este mes.</td></tr>}
        </tbody>
      </table>
      <table className="table">
        <caption>Por día de la semana</caption>
        <thead><tr><th scope="col">Día</th><th scope="col">Pomodoros</th><th scope="col">Tiempo</th></tr></thead>
        <tbody>
          {book.weekdays.map((w) => (
            <tr key={w.weekday}><th scope="row">{WEEKDAY_NAMES[w.weekday]}</th><td>{w.pomodoros}</td><td>{formatFocus(w.seconds)}</td></tr>
          ))}
        </tbody>
      </table>
      <table className="table">
        <caption>Por día</caption>
        <thead><tr><th scope="col">Fecha</th><th scope="col">Pomodoros</th><th scope="col">Tiempo</th></tr></thead>
        <tbody>
          {book.days.map((d) => (
            <tr key={d.date}><th scope="row">{dayLabel(d.date)}</th><td>{d.pomodoros}</td><td>{formatFocus(d.seconds)}</td></tr>
          ))}
          {book.days.length === 0 && <tr><td colSpan={3}>Sin datos este mes.</td></tr>}
        </tbody>
      </table>
    </div>
  )
}

/**
 * The Harvest Book (GDD §4.8): a month of focus as bales, silos or baskets, with a table view (by tag, weekday and day)
 * and a CSV export. Nothing here is a streak or a target: it only shows what you did.
 */
export function HarvestBook() {
  const [open, setOpen] = useState(false)
  /** The month asked for (null = the current one); what is shown is `book.month`, as the server resolved it. */
  const [requested, setRequested] = useState<string | null>(null)
  const [book, setBook] = useState<BookMonth | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [table, setTable] = useState(false)
  const [unit, setUnit] = useState<HarvestUnit>(loadUnit)
  const root = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    let live = true
    api.book(requested ?? '').then(
      (b) => {
        if (!live) return
        setBook(b)
        setError(null)
      },
      (e) => live && setError(e instanceof ApiError ? e.code : 'unknown'),
    )
    return () => {
      live = false
    }
  }, [open, requested])

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setOpen(false)
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open])

  const pickUnit = (u: HarvestUnit) => {
    setUnit(u)
    try {
      localStorage.setItem(UNIT_KEY, u)
    } catch {
      /* private mode */
    }
  }
  const idx = book ? book.months.indexOf(book.month) : -1
  const prev = book && idx > 0 ? book.months[idx - 1] : null
  const next = book && idx >= 0 && idx < book.months.length - 1 ? book.months[idx + 1] : null

  return (
    <div className="settings" ref={root}>
      <button type="button" className="chip chip--button" data-testid="book-button" aria-expanded={open} aria-controls="harvest-book"
        aria-label="Libro de Cosechas" onClick={() => setOpen(!open)}>
        <BookIcon />
        <span className="chip__label">Libro</span>
      </button>
      {open && (
        <section id="harvest-book" className="book" role="dialog" aria-label="Libro de Cosechas" data-testid="harvest-book">
          <header className="book__head">
            <button type="button" className="btn" data-testid="book-prev" aria-label="Mes anterior" disabled={!prev} onClick={() => prev && setRequested(prev)}>‹</button>
            <h2 data-testid="book-month" aria-live="polite">{book ? monthLabel(book.month) : 'Libro de Cosechas'}</h2>
            <button type="button" className="btn" data-testid="book-next" aria-label="Mes siguiente" disabled={!next} onClick={() => next && setRequested(next)}>›</button>
            <button type="button" className="btn" data-testid="book-close" aria-label="Cerrar el Libro" onClick={() => setOpen(false)}>✕</button>
          </header>
          {error && <p className="hint book__error" role="alert">{messageFor(error)}</p>}
          {book && (
            <>
              <p className="book__total" data-testid="book-total">
                <strong>{formatFocus(book.seconds)}</strong> de concentración · {book.pomodoros} {book.pomodoros === 1 ? 'Pomodoro' : 'Pomodoros'}
              </p>
              <div className="book__tools">
                <div role="radiogroup" aria-label="Cómo contar la cosecha" className="seg">
                  {HARVEST_UNITS.map((u) => (
                    <button key={u} type="button" role="radio" aria-checked={unit === u} className={`seg__btn${unit === u ? ' seg__btn--on' : ''}`}
                      data-testid={`book-unit-${u}`} onClick={() => pickUnit(u)}>{UNIT_NAMES[u].label}</button>
                  ))}
                </div>
                <label className="check book__table-toggle">
                  <input type="checkbox" checked={table} data-testid="book-table-toggle" onChange={(e) => setTable(e.target.checked)} />
                  <span>Ver tabla</span>
                </label>
              </div>
              {table ? <Tables book={book} /> : (
                <>
                  <Harvest seconds={book.seconds} unit={unit} />
                  <Weekdays book={book} />
                </>
              )}
              <a className="btn book__export" data-testid="book-export" href={bookExportUrl()} download="libro-de-cosechas.csv">Exportar todo a CSV</a>
            </>
          )}
          {!book && !error && <p className="hint">Abriendo el Libro…</p>}
        </section>
      )}
    </div>
  )
}
