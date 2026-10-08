import { describe, expect, it } from 'vitest'
import { DEFAULT_STATIONS, parseVideoId } from './stations'
import { isFatalError } from './youtube'

const ID = 'jfKfPfyJRdk'

describe('parseVideoId', () => {
  it.each([
    [ID, ID],
    [`  ${ID}  `, ID],
    [`https://www.youtube.com/watch?v=${ID}`, ID],
    [`https://youtube.com/watch?v=${ID}&t=30s`, ID],
    [`https://www.youtube.com/watch?feature=share&v=${ID}`, ID],
    [`www.youtube.com/watch?v=${ID}`, ID],
    [`https://m.youtube.com/watch?v=${ID}`, ID],
    [`https://music.youtube.com/watch?v=${ID}`, ID],
    [`https://youtu.be/${ID}`, ID],
    [`https://youtu.be/${ID}?si=abc`, ID],
    [`https://www.youtube.com/live/${ID}`, ID],
    [`https://www.youtube.com/embed/${ID}`, ID],
    [`https://www.youtube-nocookie.com/embed/${ID}`, ID],
    [`https://www.youtube.com/shorts/${ID}`, ID],
  ])('reads %s', (input, out) => expect(parseVideoId(input)).toBe(out))

  it.each([
    '', '   ', 'hello', 'jfKfPfyJRd', 'jfKfPfyJRdkk', 'jfKfPfyJRd!',
    'https://example.com/watch?v=jfKfPfyJRdk',
    'https://youtube.com.evil.com/watch?v=jfKfPfyJRdk',
    'https://evil.com/youtu.be/jfKfPfyJRdk',
    'javascript:alert(1)',
    'https://www.youtube.com/playlist?list=PLabcdefghijk',
    'https://www.youtube.com/watch?v=short',
    'https://www.youtube.com/watch?v=<script>x</script>',
    'ftp://youtu.be/jfKfPfyJRdk',
  ])('rejects %j', (input) => expect(parseVideoId(input)).toBeNull())
})

describe('DEFAULT_STATIONS', () => {
  it('are valid, distinct video ids', () => {
    for (const s of DEFAULT_STATIONS) expect(parseVideoId(s.id)).toBe(s.id)
    expect(new Set(DEFAULT_STATIONS.map((s) => s.id)).size).toBe(DEFAULT_STATIONS.length)
  })
})

describe('isFatalError', () => {
  it.each([2, 5, 100, 101, 150])('%s means the video will not play', (c) => expect(isFatalError(c)).toBe(true))
  it.each([0, 1, 3, 200])('%s is not treated as fatal', (c) => expect(isFatalError(c)).toBe(false))
})
