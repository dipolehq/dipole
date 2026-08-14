import { useState, useEffect, useRef, useCallback, useImperativeHandle, forwardRef } from 'react'

export interface RobotHandle {
  say: (text: string) => void
  setMood: (mood: string) => void
  turnAway: (on: boolean) => void
  followTyping: (value: string) => void
  hype: (on: boolean) => void
  press: () => void
  release: () => void
  success: (message: string) => void
  setPasswordStrength: (score: number, label: string) => void
}

/**
 * Reusable animated robot head that sits above any auth form card.
 * Exposes a ref handle for the parent form to control mood, speech, and eye tracking.
 */
export const RobotGuard = forwardRef<RobotHandle>(function RobotGuard(_props, ref) {
  const [mood, setMoodState] = useState<string>('idle')
  const [bubbleMsg, setBubbleMsg] = useState("Hi. I'm Volt. I guard this form.")
  const [isTurned, setIsTurned] = useState(false)
  const [isHyped, setIsHyped] = useState(false)
  const [isPressed, setIsPressed] = useState(false)
  const [isSpinning, setIsSpinning] = useState(false)
  const [eyePos, setEyePos] = useState({ x: 0, y: 0 })
  const [headTilt, setHeadTilt] = useState({ ry: 0, rx: 0 })
  const [meterLvl, setMeterLvl] = useState(0)
  const [meterLabel, setMeterLabel] = useState('NOT LOOKING')
  const [popBubble, setPopBubble] = useState(false)

  const robotRef = useRef<HTMLDivElement>(null)
  const lastSaidRef = useRef('')
  const pressTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined)
  const doneRef = useRef(false)

  const say = useCallback((text: string) => {
    if (text === lastSaidRef.current) return
    lastSaidRef.current = text
    setBubbleMsg(text)
    setPopBubble(false)
    requestAnimationFrame(() => setPopBubble(true))
  }, [])

  const followTyping = useCallback((value: string) => {
    const ratio = Math.min(value.length / 22, 1)
    setEyePos({ x: -6 + 12 * ratio, y: 5 })
    setHeadTilt({ ry: -5 + 10 * ratio, rx: -8 })
  }, [])

  // Expose handle to parent
  useImperativeHandle(ref, () => ({
    say,
    setMood: (m: string) => { if (!doneRef.current) setMoodState(m) },
    turnAway: (on: boolean) => {
      setIsTurned(on)
      if (on) { setEyePos({ x: 0, y: 0 }); setHeadTilt({ ry: 0, rx: 0 }) }
    },
    followTyping,
    hype: (on: boolean) => {
      if (doneRef.current) return
      setIsHyped(on)
      if (on) setIsTurned(false)
    },
    press: () => {
      clearTimeout(pressTimerRef.current)
      setIsPressed(true)
      setMoodState('pressed')
    },
    release: () => {
      clearTimeout(pressTimerRef.current)
      pressTimerRef.current = setTimeout(() => {
        setIsPressed(false)
      }, 340)
    },
    success: (message: string) => {
      doneRef.current = true
      setIsTurned(false)
      setIsHyped(false)
      setMoodState('success')
      say(message)
      setEyePos({ x: 0, y: 0 })
      setHeadTilt({ ry: 0, rx: 0 })
      setIsSpinning(true)
      setTimeout(() => setIsSpinning(false), 950)
    },
    setPasswordStrength: (score: number, label: string) => {
      setMeterLvl(score)
      setMeterLabel(label)
    },
  }), [say, followTyping])

  // Blinking
  useEffect(() => {
    let timeout: ReturnType<typeof setTimeout>
    function blinkLoop() {
      timeout = setTimeout(() => {
        if (mood !== 'success' && !isTurned) {
          const eyesEl = document.getElementById('robot-eyes')
          if (eyesEl) {
            eyesEl.classList.add('blink')
            setTimeout(() => eyesEl.classList.remove('blink'), 150)
          }
        }
        blinkLoop()
      }, 2600 + Math.random() * 2600)
    }
    blinkLoop()
    return () => clearTimeout(timeout)
  }, [mood, isTurned])

  // Mouse follow
  useEffect(() => {
    function handleMouseMove(e: MouseEvent) {
      const active = document.activeElement
      if (doneRef.current || (active && active.tagName === 'INPUT')) return
      const robot = robotRef.current
      if (!robot) return
      const rect = robot.getBoundingClientRect()
      const cx = rect.left + rect.width / 2
      const cy = rect.top + rect.height / 2
      const dx = Math.max(-1, Math.min(1, (e.clientX - cx) / 260))
      const dy = Math.max(-1, Math.min(1, (e.clientY - cy) / 260))
      setEyePos({ x: dx * 7, y: dy * 6 })
      if (!isTurned) setHeadTilt({ ry: dx * 12, rx: -dy * 9 })
    }
    document.addEventListener('mousemove', handleMouseMove)
    return () => document.removeEventListener('mousemove', handleMouseMove)
  }, [isTurned])

  // Password meter (exposed via mood changes from parent)
  useEffect(() => {
    if (mood === 'shy') {
      setMeterLabel('NOT LOOKING')
      setMeterLvl(0)
    }
  }, [mood])

  const robotClasses = [
    'robot',
    isHyped ? 'is-hyped' : '',
    isPressed ? 'is-pressed' : '',
    isTurned ? 'is-turned' : '',
    isSpinning ? 'is-spinning' : '',
  ].filter(Boolean).join(' ')

  return (
    <div className={robotClasses} ref={robotRef} data-mood={mood}>
      <div className={`bubble ${popBubble ? 'pop' : ''}`} role="status" aria-live="polite">
        <span>{bubbleMsg}</span>
      </div>

      <div className="antenna" aria-hidden="true">
        <span className="antenna-rod" />
        <span className="antenna-tip" />
      </div>

      <div
        className="head3d"
        aria-hidden="true"
        style={{
          '--ry': `${headTilt.ry}deg`,
          '--rx': `${headTilt.rx}deg`,
          '--hy': isHyped ? '5px' : isPressed ? '9px' : '0px',
          '--hsc': isPressed ? '0.94' : isHyped ? '1.04' : '1',
        } as React.CSSProperties}
      >
        <div className="head">
          <span className="ear ear--l" />
          <span className="ear ear--r" />

          <div className="face face--front">
            <div className="visor">
              <div
                className="eyes"
                id="robot-eyes"
                style={{ '--lx': `${eyePos.x}px`, '--ly': `${eyePos.y}px` } as React.CSSProperties}
              >
                <span className="eye eye--l" />
                <span className="eye eye--r" />
              </div>
              <span className="cheek cheek--l" />
              <span className="cheek cheek--r" />
              <span className="mouth" />
            </div>
          </div>

          <div className="face face--back">
            <div className="panel">
              <span className="panel-lights"><i /><i /><i /></span>
              <div className="meter" data-lvl={meterLvl}>
                {[0, 1, 2, 3].map((i) => (
                  <i key={i} className={i < meterLvl ? 'on' : ''} />
                ))}
              </div>
              <p className="panel-label">{meterLabel}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
})
