import { useEffect, useRef } from 'react'

// Simplex Noise implementation
class SimplexNoise {
  private grad3 = [[1,1,0],[-1,1,0],[1,-1,0],[-1,-1,0],[1,0,1],[-1,0,1],[1,0,-1],[-1,0,-1],[0,1,1],[0,-1,1],[0,1,-1],[0,-1,-1]]
  private perm: number[] = []

  constructor() {
    const p: number[] = []
    for (let i = 0; i < 256; i++) p[i] = Math.floor(Math.random() * 256)
    for (let i = 0; i < 512; i++) this.perm[i] = p[i & 255]!
  }

  private dot(g: number[], x: number, y: number) { return g[0]! * x + g[1]! * y }

  noise(xin: number, yin: number): number {
    const F2 = 0.5 * (Math.sqrt(3.0) - 1.0)
    const s = (xin + yin) * F2
    const i = Math.floor(xin + s)
    const j = Math.floor(yin + s)
    const G2 = (3.0 - Math.sqrt(3.0)) / 6.0
    const t = (i + j) * G2
    const x0 = xin - (i - t)
    const y0 = yin - (j - t)
    const i1 = x0 > y0 ? 1 : 0
    const j1 = x0 > y0 ? 0 : 1
    const x1 = x0 - i1 + G2
    const y1 = y0 - j1 + G2
    const x2 = x0 - 1.0 + 2.0 * G2
    const y2 = y0 - 1.0 + 2.0 * G2
    const ii = i & 255
    const jj = j & 255
    const gi0 = this.perm[ii + this.perm[jj]!]! % 12
    const gi1 = this.perm[ii + i1 + this.perm[jj + j1]!]! % 12
    const gi2 = this.perm[ii + 1 + this.perm[jj + 1]!]! % 12
    let n0: number, n1: number, n2: number
    let t0 = 0.5 - x0 * x0 - y0 * y0
    if (t0 < 0) n0 = 0.0
    else { t0 *= t0; n0 = t0 * t0 * this.dot(this.grad3[gi0]!, x0, y0) }
    let t1 = 0.5 - x1 * x1 - y1 * y1
    if (t1 < 0) n1 = 0.0
    else { t1 *= t1; n1 = t1 * t1 * this.dot(this.grad3[gi1]!, x1, y1) }
    let t2 = 0.5 - x2 * x2 - y2 * y2
    if (t2 < 0) n2 = 0.0
    else { t2 *= t2; n2 = t2 * t2 * this.dot(this.grad3[gi2]!, x2, y2) }
    return 70.0 * (n0 + n1 + n2)
  }
}

const GRID_SIZE = 3
const FLOW_SPEED = 0.005
const CLOUD_ZOOM = 0.0018
const MOUSE_RADIUS = 350
const SCATTER_FORCE = 120
const COLORS = ['#222222', '#555555', '#999999', '#ffffff']

export function CloudCanvas() {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const mouseRef = useRef({ x: -500, y: -500 })
  const timeRef = useRef(0)
  const animRef = useRef<number>(0)

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const noiseGen = new SimplexNoise()

    function resize() {
      canvas!.width = window.innerWidth
      canvas!.height = window.innerHeight
    }

    function fbm(x: number, y: number): number {
      let total = 0, amp = 0.6, freq = 1.0, max = 0
      for (let i = 0; i < 5; i++) {
        total += noiseGen.noise(x * CLOUD_ZOOM * freq, y * CLOUD_ZOOM * freq) * amp
        max += amp
        amp *= 0.5
        freq *= 2.0
      }
      return Math.pow((total / max) + 0.5, 2.2)
    }

    function draw() {
      const w = canvas!.width
      const h = canvas!.height
      const mouse = mouseRef.current
      const time = timeRef.current

      ctx!.fillStyle = '#050505'
      ctx!.fillRect(0, 0, w, h)

      for (let y = 0; y < h; y += GRID_SIZE) {
        for (let x = 0; x < w; x += GRID_SIZE) {
          const n = fbm(x + time * 50, y + time * 30)
          if (n < 0.12) continue

          let drawX = x
          let drawY = y

          const dx = x - mouse.x
          const dy = y - mouse.y
          const dist = Math.sqrt(dx * dx + dy * dy)

          if (dist < MOUSE_RADIUS) {
            const f = Math.pow((MOUSE_RADIUS - dist) / MOUSE_RADIUS, 2)
            const a = Math.atan2(dy, dx)
            const p = f * SCATTER_FORCE
            const j = (Math.random() - 0.5) * 30 * f
            drawX += Math.cos(a) * p + j
            drawY += Math.sin(a) * p + j
          }

          const intensity = (n - 0.12) / 0.88
          const cIdx = intensity > 0.75 ? 3 : intensity > 0.5 ? 2 : intensity > 0.25 ? 1 : 0
          ctx!.fillStyle = COLORS[cIdx]!

          if (Math.random() < intensity * 2.5) {
            ctx!.fillRect(drawX, drawY, GRID_SIZE - 1, GRID_SIZE - 1)
          }
        }
      }

      timeRef.current += FLOW_SPEED
      animRef.current = requestAnimationFrame(draw)
    }

    function onMouseMove(e: MouseEvent) { mouseRef.current = { x: e.clientX, y: e.clientY } }
    function onMouseOut() { mouseRef.current = { x: -500, y: -500 } }
    function onTouchMove(e: TouchEvent) {
      if (e.touches[0]) mouseRef.current = { x: e.touches[0].clientX, y: e.touches[0].clientY }
    }
    function onTouchEnd() { mouseRef.current = { x: -500, y: -500 } }

    resize()
    window.addEventListener('resize', resize)
    window.addEventListener('mousemove', onMouseMove)
    window.addEventListener('mouseout', onMouseOut)
    window.addEventListener('touchmove', onTouchMove, { passive: false })
    window.addEventListener('touchend', onTouchEnd)

    animRef.current = requestAnimationFrame(draw)

    return () => {
      cancelAnimationFrame(animRef.current)
      window.removeEventListener('resize', resize)
      window.removeEventListener('mousemove', onMouseMove)
      window.removeEventListener('mouseout', onMouseOut)
      window.removeEventListener('touchmove', onTouchMove)
      window.removeEventListener('touchend', onTouchEnd)
    }
  }, [])

  return <canvas ref={canvasRef} className="fixed inset-0 w-full h-full" />
}
