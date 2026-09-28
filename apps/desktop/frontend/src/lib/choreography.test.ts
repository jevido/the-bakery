import { describe, expect, test } from 'bun:test'
import { newWorld, step, type Input, type Room, type RunIn, type World } from './choreography'

const room: Room = {
  width: 800,
  height: 500,
  desk: { x: 400, y: 440 },
  benches: { coding: { x: 100, y: 80 }, research: { x: 300, y: 80 }, '': { x: 500, y: 80 } },
  wander: { x: 50, y: 200, w: 700, h: 180 },
}

function run(world: World, runs: RunIn[], from: number, ms: number, agents = ['moss', 'pip'], quiet = false) {
  let w = world
  for (let t = from; t < from + ms; t += 50) {
    const input: Input = { now: t, agents, runs, room, quiet }
    w = step(w, input, 50)
  }
  return w
}

const moss = (status: string, id = 'r1'): RunIn => ({ id, agent: 'moss', status, workType: 'coding', notes: 'Done: moved to Review' })

describe('choreography', () => {
  test('agents dilly-dally while nothing runs', () => {
    const w = run(newWorld(room), [], 0, 10_000)
    expect(w.figures.moss.state).toBe('wander')
    expect(w.supervisor.state).toBe('desk')
  })

  test('a new run: the supervisor hands it out, then the agent goes to its bench', () => {
    let w = run(newWorld(room), [], 0, 1000)
    w = run(w, [moss('running')], 1000, 200)
    expect(w.figures.moss.state).toBe('waiting')
    expect(['to_agent', 'handing']).toContain(w.supervisor.state)
    w = run(w, [moss('running')], 1200, 30_000)
    expect(w.figures.moss.state).toBe('working')
    expect(w.figures.moss.pos).toEqual(room.benches.coding)
    expect(w.supervisor.state).toBe('desk')
  })

  test('the run ends: the agent brings its notes back, then wanders again', () => {
    let w = run(newWorld(room), [], 0, 1000)
    w = run(w, [moss('running')], 1000, 30_000)
    w = run(w, [moss('succeeded')], 31_000, 200)
    expect(w.figures.moss.state).toBe('to_supervisor')
    expect(w.figures.moss.notes).toBe('Done: moved to Review')
    w = run(w, [moss('succeeded')], 31_200, 20_000)
    expect(['handing_in', 'wander']).toContain(w.figures.moss.state)
    w = run(w, [moss('succeeded')], 51_200, 5_000)
    expect(w.figures.moss.state).toBe('wander')
  })

  test('two runs starting at once are handed out one after the other', () => {
    let w = run(newWorld(room), [], 0, 1000)
    const both: RunIn[] = [moss('running'), { id: 'r2', agent: 'pip', status: 'running', workType: 'research', notes: '' }]
    w = run(w, both, 1000, 100)
    expect(w.supervisor.current).not.toBeNull()
    expect(w.supervisor.queue.length).toBe(1)
    w = run(w, both, 1100, 60_000)
    expect(w.figures.moss.state).toBe('working')
    expect(w.figures.pip.state).toBe('working')
    expect(w.figures.pip.pos).toEqual(room.benches.research)
  })

  test('a run already going when the canvas opens is at its bench at once', () => {
    const w = run(newWorld(room), [moss('running')], 0, 50)
    expect(w.figures.moss.state).toBe('working')
    expect(w.supervisor.state).toBe('desk')
  })

  test('quiet colony: nobody moves; workers at their bench, the rest by the desk', () => {
    let w = run(newWorld(room), [], 0, 1000, ['moss', 'pip'], true)
    const pip = { ...w.figures.pip.pos }
    w = run(w, [moss('running')], 1000, 5000, ['moss', 'pip'], true)
    expect(w.figures.moss.pos).toEqual(room.benches.coding)
    expect(w.figures.pip.pos).toEqual(pip)
    expect(w.supervisor.pos).toEqual(room.desk)
    w = run(w, [moss('failed')], 6000, 1000, ['moss', 'pip'], true)
    expect(w.figures.moss.state).toBe('wander')
    expect(w.figures.moss.bubbleUntil).toBe(0)
  })

  test('the same inputs give the same world', () => {
    const a = run(newWorld(room), [moss('running')], 0, 20_000)
    const b = run(newWorld(room), [moss('running')], 0, 20_000)
    expect(a).toEqual(b)
  })
})

describe('the supervisor and the chat', () => {
  test('it thinks at the desk while the chat waits for an answer', () => {
    let w = newWorld(room)
    for (let t = 0; t < 2000; t += 50) w = step(w, { now: t, agents: ['moss'], runs: [], room, quiet: false, thinking: true }, 50)
    expect(w.supervisor.thinking).toBe(true)
    expect(w.supervisor.pos).toEqual(room.desk)
    w = step(w, { now: 2050, agents: ['moss'], runs: [], room, quiet: false, thinking: false }, 50)
    expect(w.supervisor.thinking).toBe(false)
  })

  test('a task to hand out comes before thinking', () => {
    let w = run(newWorld(room), [], 0, 1000)
    for (let t = 1000; t < 1500; t += 50) w = step(w, { now: t, agents: ['moss', 'pip'], runs: [moss('running')], room, quiet: false, thinking: true }, 50)
    expect(w.supervisor.state).not.toBe('desk')
    expect(w.supervisor.thinking).toBe(false)
  })

  test('quiet colony: no thinking either', () => {
    const w = step(newWorld(room), { now: 0, agents: [], runs: [], room, quiet: true, thinking: true }, 50)
    expect(w.supervisor.thinking).toBe(false)
  })
})
