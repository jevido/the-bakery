// What the figures on Work mode's canvas do, as a pure state machine. It
// acts out the runs that already happen and nothing else: an agent
// dilly-dallies until the supervisor walks up and hands it a task, walks to
// the bench of the task's work type and works while its run goes, then
// brings its notes back to the supervisor and wanders again. step() takes
// the world, what the app knows now, and the time passed, and returns the
// next world; the same inputs always give the same world.

export type Vec = { x: number; y: number }

// The room: where things stand. Positions are figure positions (their feet).
export type Room = {
  width: number
  height: number
  desk: Vec // where the supervisor waits
  benches: Record<string, Vec> // work type key → where its worker stands; '' for any other work
  wander: { x: number; y: number; w: number; h: number } // the floor agents stroll on
}

export type RunIn = {
  id: string
  agent: string
  status: string // running, succeeded, failed, stopped
  workType: string
  notes: string // what the agent brings back when the run ends
}

export type Input = {
  now: number // ms
  agents: string[] // agents on the board, in order
  runs: RunIn[] // this board's runs on this machine
  room: Room
  quiet: boolean // quiet colony or reduced motion: everyone stands still in place
  thinking?: boolean // the supervisor is writing an answer in the chat
}

export type AgentState = 'wander' | 'waiting' | 'to_bench' | 'working' | 'to_supervisor' | 'handing_in'
export type SupervisorState = 'desk' | 'to_agent' | 'handing' | 'back'

export type Figure = {
  slug: string
  pos: Vec
  target: Vec | null
  state: AgentState
  runId: string | null
  workType: string
  // wandering: a pause before the next stroll ends at restUntil
  restUntil: number
  // handing in: the bubble shows until bubbleUntil
  notes: string
  bubbleUntil: number
  rng: number
}

export type Supervisor = {
  pos: Vec
  target: Vec | null
  state: SupervisorState
  queue: string[] // agents waiting for their task, in order
  current: string | null // the agent being handed a task
  until: number // handing ends
  // thinking: standing at the desk over the chat's answer
  thinking: boolean
}

export type World = {
  started: boolean
  figures: Record<string, Figure>
  supervisor: Supervisor
  // What the world has already acted out for each run.
  runs: Record<string, 'handed' | 'ended'>
}

export const AGENT_SPEED = 55 // px a second while strolling; errands are faster
export const SUPERVISOR_SPEED = 180
export const HAND_OVER_MS = 900
export const BUBBLE_MS = 3000

export function newWorld(room: Room): World {
  return {
    started: false,
    figures: {},
    supervisor: { pos: { ...room.desk }, target: null, state: 'desk', queue: [], current: null, until: 0, thinking: false },
    runs: {},
  }
}

// seedOf turns an agent's slug into its own random stream.
function seedOf(s: string): number {
  let h = 0x811c9dc5
  for (const ch of s) {
    h ^= ch.charCodeAt(0)
    h = Math.imul(h, 0x01000193) >>> 0
  }
  return h
}

// next draws a number in [0, 1) and the stream's next state (mulberry32).
function next(state: number): [number, number] {
  const a = (state + 0x6d2b79f5) >>> 0
  let t = a
  t = Math.imul(t ^ (t >>> 15), t | 1)
  t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
  return [((t ^ (t >>> 14)) >>> 0) / 4294967296, a]
}

function dist(a: Vec, b: Vec): number {
  return Math.hypot(a.x - b.x, a.y - b.y)
}

// walk moves pos toward target at speed for dt ms; arrived says it is there.
function walk(pos: Vec, target: Vec, speed: number, dt: number): { pos: Vec; arrived: boolean } {
  const d = dist(pos, target)
  const stepLen = (speed * dt) / 1000
  if (d <= stepLen || d < 0.5) return { pos: { ...target }, arrived: true }
  return { pos: { x: pos.x + ((target.x - pos.x) / d) * stepLen, y: pos.y + ((target.y - pos.y) / d) * stepLen }, arrived: false }
}

// home is where an agent stands when nothing moves: in a line beside the desk.
export function home(room: Room, index: number): Vec {
  return { x: room.desk.x + 40 + index * 28, y: room.desk.y }
}

function benchOf(room: Room, workType: string): Vec {
  return room.benches[workType] ?? room.benches[''] ?? room.desk
}

// Beside the supervisor, where an agent hands in its notes.
function besideSupervisor(sup: Supervisor): Vec {
  return { x: sup.pos.x + 22, y: sup.pos.y }
}

function spawn(room: Room, slug: string, index: number): Figure {
  let rng = seedOf(slug)
  let r1: number
  let r2: number
  ;[r1, rng] = next(rng)
  ;[r2, rng] = next(rng)
  const pos = { x: room.wander.x + r1 * room.wander.w, y: room.wander.y + r2 * room.wander.h }
  return {
    slug, pos: index >= 0 ? pos : pos, target: null, state: 'wander', runId: null, workType: '',
    restUntil: 0, notes: '', bubbleUntil: 0, rng,
  }
}

export function step(world: World, input: Input, dt: number): World {
  const { room, now, quiet } = input
  const figures: Record<string, Figure> = {}
  const runs = { ...world.runs }
  const sup: Supervisor = { ...world.supervisor, queue: [...world.supervisor.queue] }

  // Every agent on the board, and any with a run here, has a figure.
  const slugs = [...input.agents]
  for (const r of input.runs) if (!slugs.includes(r.agent)) slugs.push(r.agent)
  slugs.forEach((slug, i) => {
    figures[slug] = world.figures[slug] ? { ...world.figures[slug] } : spawn(room, slug, i)
  })

  // The latest run of each agent decides what it should be doing.
  const latest = new Map<string, RunIn>()
  for (const r of input.runs) latest.set(r.agent, r)

  for (const [slug, r] of latest) {
    const f = figures[slug]
    if (r.status === 'running' && !runs[r.id]) {
      runs[r.id] = 'handed'
      f.runId = r.id
      f.workType = r.workType
      f.notes = ''
      if (!world.started || quiet) {
        // Already running when the canvas opened, or nothing may move:
        // straight to the bench.
        f.state = 'working'
        f.pos = { ...benchOf(room, r.workType) }
        f.target = null
      } else {
        f.state = 'waiting'
        f.target = null
        if (!sup.queue.includes(slug) && sup.current !== slug) sup.queue.push(slug)
      }
    } else if (r.status !== 'running' && runs[r.id] === 'handed') {
      runs[r.id] = 'ended'
      sup.queue = sup.queue.filter((s) => s !== slug)
      if (sup.current === slug) {
        sup.current = null
        sup.state = 'back'
        sup.target = { ...room.desk }
      }
      f.runId = r.id
      f.notes = r.notes
      if (quiet) {
        f.state = 'wander'
        f.target = null
      } else {
        f.state = 'to_supervisor'
        f.target = besideSupervisor(sup)
      }
    } else if (r.status !== 'running' && !runs[r.id]) {
      // Ended before the canvas saw it start: nothing to act out.
      runs[r.id] = 'ended'
    }
  }

  // The supervisor: hand out the waiting agents' tasks one after another.
  if (!quiet) {
    if (sup.state === 'desk' && sup.queue.length) {
      sup.current = sup.queue.shift()!
      sup.state = 'to_agent'
    }
    if (sup.state === 'to_agent' && sup.current) {
      const agent = figures[sup.current]
      sup.target = { x: agent.pos.x - 18, y: agent.pos.y }
      const w = walk(sup.pos, sup.target, SUPERVISOR_SPEED, dt)
      sup.pos = w.pos
      if (w.arrived) {
        sup.state = 'handing'
        sup.until = now + HAND_OVER_MS
      }
    } else if (sup.state === 'handing' && sup.current && now >= sup.until) {
      const agent = figures[sup.current]
      agent.state = 'to_bench'
      agent.target = { ...benchOf(room, agent.workType) }
      sup.current = null
      sup.state = sup.queue.length ? 'desk' : 'back'
      sup.target = { ...room.desk }
    } else if (sup.state === 'back') {
      if (sup.queue.length) sup.state = 'desk'
      else {
        const w = walk(sup.pos, room.desk, SUPERVISOR_SPEED, dt)
        sup.pos = w.pos
        if (w.arrived) {
          sup.state = 'desk'
          sup.target = null
        }
      }
    }
    // The desk moves with the room when the window is resized.
    if (sup.state === 'desk') sup.pos = { ...room.desk }
  } else {
    sup.pos = { ...room.desk }
    sup.state = 'desk'
    sup.queue = []
    sup.current = null
    sup.target = null
  }

  // The agents.
  slugs.forEach((slug, i) => {
    const f = figures[slug]
    if (quiet) {
      f.target = null
      f.bubbleUntil = 0
      if (f.state === 'working') f.pos = { ...benchOf(room, f.workType) }
      else {
        f.state = f.state === 'waiting' || f.state === 'to_bench' ? 'working' : 'wander'
        f.pos = f.state === 'working' ? { ...benchOf(room, f.workType) } : home(room, i)
      }
      return
    }
    switch (f.state) {
      case 'wander': {
        if (f.target) {
          const w = walk(f.pos, f.target, AGENT_SPEED, dt)
          f.pos = w.pos
          if (w.arrived) {
            let r: number
            ;[r, f.rng] = next(f.rng)
            f.target = null
            f.restUntil = now + 1000 + r * 3000
          }
        } else if (now >= f.restUntil) {
          let rx: number
          let ry: number
          ;[rx, f.rng] = next(f.rng)
          ;[ry, f.rng] = next(f.rng)
          f.target = { x: room.wander.x + rx * room.wander.w, y: room.wander.y + ry * room.wander.h }
        }
        break
      }
      case 'to_bench':
      case 'to_supervisor': {
        if (f.state === 'to_supervisor') f.target = besideSupervisor(sup)
        const w = walk(f.pos, f.target!, AGENT_SPEED * 3, dt)
        f.pos = w.pos
        if (w.arrived) {
          if (f.state === 'to_bench') f.state = 'working'
          else {
            f.state = 'handing_in'
            f.bubbleUntil = now + BUBBLE_MS
          }
          f.target = null
        }
        break
      }
      case 'handing_in':
        if (now >= f.bubbleUntil) {
          f.state = 'wander'
          f.runId = null
          f.restUntil = now + 500
        }
        break
      case 'waiting':
      case 'working':
        break
    }
  })

  // Handing out tasks comes first; the supervisor thinks at the desk when
  // nobody is waiting.
  sup.thinking = !quiet && !!input.thinking && sup.state === 'desk' && sup.queue.length === 0
  return { started: true, figures, supervisor: sup, runs }
}

// moving says whether anything on the canvas will move without new input,
// so the drawing loop can sleep when it does not.
export function moving(world: World, now: number): boolean {
  const s = world.supervisor
  if (s.state !== 'desk' || s.queue.length || s.thinking) return true
  return Object.values(world.figures).some(
    (f) => f.state === 'wander' || f.state === 'to_bench' || f.state === 'to_supervisor' || (f.state === 'handing_in' && now < f.bubbleUntil + 100),
  )
}
