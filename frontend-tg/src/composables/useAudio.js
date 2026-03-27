// Web Audio API 音效引擎 - 炸金花
let audioCtx = null
let bgmGain = null
let bgmPlaying = false
let bgmTimer = null
let actionGain = null
let actionPlaying = false
let actionTimer = null

function getCtx() {
  if (!audioCtx) {
    audioCtx = new (window.AudioContext || window.webkitAudioContext)()
  }
  if (audioCtx.state === 'suspended') audioCtx.resume()
  return audioCtx
}

function createTone(ctx, freq, type, filterFreq) {
  const osc = ctx.createOscillator()
  const gain = ctx.createGain()
  const filter = ctx.createBiquadFilter()
  osc.type = type
  osc.frequency.value = freq
  filter.type = 'lowpass'
  filter.frequency.value = filterFreq || 2000
  osc.connect(filter)
  filter.connect(gain)
  return { osc, gain, filter }
}

// 播放音效
export function playSound(type) {
  try {
    const ctx = getCtx()
    switch (type) {
      case 'chip': {
        const t = ctx.currentTime
        for (let i = 0; i < 3; i++) {
          const { osc, gain } = createTone(ctx, 2500 + i * 800, 'sine', 4000)
          gain.connect(ctx.destination)
          gain.gain.setValueAtTime(0.08, t + i * 0.03)
          gain.gain.exponentialRampToValueAtTime(0.001, t + i * 0.03 + 0.08)
          osc.start(t + i * 0.03)
          osc.stop(t + i * 0.03 + 0.08)
        }
        break
      }
      case 'card': {
        const t = ctx.currentTime
        const bufferSize = ctx.sampleRate * 0.06
        const buffer = ctx.createBuffer(1, bufferSize, ctx.sampleRate)
        const data = buffer.getChannelData(0)
        for (let i = 0; i < bufferSize; i++) {
          data[i] = (Math.random() * 2 - 1) * Math.exp(-i / (bufferSize * 0.15))
        }
        const noise = ctx.createBufferSource()
        noise.buffer = buffer
        const filter = ctx.createBiquadFilter()
        filter.type = 'bandpass'
        filter.frequency.value = 3000
        filter.Q.value = 2
        const g = ctx.createGain()
        g.gain.value = 0.25
        noise.connect(filter)
        filter.connect(g)
        g.connect(ctx.destination)
        noise.start(t)
        break
      }
      case 'win': {
        const pentatonic = [523, 587, 659, 784, 880, 1047]
        const t = ctx.currentTime
        pentatonic.forEach((freq, i) => {
          const { osc, gain } = createTone(ctx, freq, 'sine', 3000)
          gain.connect(ctx.destination)
          const start = t + i * 0.1
          gain.gain.setValueAtTime(0, start)
          gain.gain.linearRampToValueAtTime(0.12, start + 0.03)
          gain.gain.exponentialRampToValueAtTime(0.001, start + 0.35)
          osc.start(start)
          osc.stop(start + 0.35)
        })
        const { osc: last, gain: lg } = createTone(ctx, 1047, 'sine', 2000)
        lg.connect(ctx.destination)
        const end = t + 0.6
        lg.gain.setValueAtTime(0, end)
        lg.gain.linearRampToValueAtTime(0.1, end + 0.05)
        lg.gain.exponentialRampToValueAtTime(0.001, end + 0.8)
        last.start(end)
        last.stop(end + 0.8)
        return
      }
      case 'lose': {
        const t = ctx.currentTime
        const { osc, gain } = createTone(ctx, 350, 'triangle', 800)
        gain.connect(ctx.destination)
        osc.frequency.exponentialRampToValueAtTime(120, t + 0.5)
        gain.gain.setValueAtTime(0.1, t)
        gain.gain.exponentialRampToValueAtTime(0.001, t + 0.5)
        osc.start(t)
        osc.stop(t + 0.5)
        break
      }
      case 'fold': {
        const t = ctx.currentTime
        const { osc, gain } = createTone(ctx, 500, 'sine', 1200)
        gain.connect(ctx.destination)
        osc.frequency.exponentialRampToValueAtTime(200, t + 0.15)
        gain.gain.setValueAtTime(0.06, t)
        gain.gain.exponentialRampToValueAtTime(0.001, t + 0.15)
        osc.start(t)
        osc.stop(t + 0.15)
        break
      }
      case 'click': {
        const t = ctx.currentTime
        const { osc, gain } = createTone(ctx, 800, 'sine', 2000)
        gain.connect(ctx.destination)
        gain.gain.setValueAtTime(0.06, t)
        gain.gain.exponentialRampToValueAtTime(0.001, t + 0.04)
        osc.start(t)
        osc.stop(t + 0.04)
        break
      }
    }
  } catch (e) {}
}

// ============================================================
// 令人兴奋的赌场风格 BGM - 快节奏、激昂、中国风+赌场能量
// 120-130 BPM，大调旋律，琶音+节奏驱动
// ============================================================
export function startBGM() {
  if (bgmPlaying) return
  try {
    const ctx = getCtx()
    bgmGain = ctx.createGain()
    bgmGain.gain.value = 0.045
    bgmGain.connect(ctx.destination)
    bgmPlaying = true
    scheduleCasinoBGM(ctx)
  } catch (e) {}
}

function scheduleCasinoBGM(ctx) {
  if (!bgmPlaying) return
  const t = ctx.currentTime + 0.05
  const bpm = 125
  const beat = 60 / bpm // ~0.48s per beat

  // ---- 快速驱动鼓点 - 四拍底鼓+反拍军鼓+16分音符踩镲 ----
  for (let i = 0; i < 16; i++) {
    const bt = t + i * beat

    // Kick on every beat
    const kickOsc = ctx.createOscillator()
    const kickG = ctx.createGain()
    kickOsc.type = 'sine'
    kickOsc.frequency.setValueAtTime(140, bt)
    kickOsc.frequency.exponentialRampToValueAtTime(45, bt + 0.12)
    kickOsc.connect(kickG)
    kickG.connect(bgmGain)
    kickG.gain.setValueAtTime(2.2, bt)
    kickG.gain.exponentialRampToValueAtTime(0.001, bt + 0.18)
    kickOsc.start(bt)
    kickOsc.stop(bt + 0.18)

    // Snare on 2 & 4 (offbeats)
    if (i % 2 === 1) {
      const len = ctx.sampleRate * 0.07
      const buf = ctx.createBuffer(1, len, ctx.sampleRate)
      const ch = buf.getChannelData(0)
      for (let j = 0; j < len; j++) ch[j] = (Math.random() * 2 - 1) * Math.exp(-j / (len * 0.18))
      const src = ctx.createBufferSource()
      src.buffer = buf
      const f = ctx.createBiquadFilter()
      f.type = 'highpass'
      f.frequency.value = 1200
      const g = ctx.createGain()
      g.gain.value = 1.3
      src.connect(f)
      f.connect(g)
      g.connect(bgmGain)
      src.start(bt)
    }

    // Hi-hat 16ths (two per beat)
    for (let h = 0; h < 2; h++) {
      const ht = bt + h * beat * 0.5
      const len2 = ctx.sampleRate * 0.015
      const buf2 = ctx.createBuffer(1, len2, ctx.sampleRate)
      const ch2 = buf2.getChannelData(0)
      for (let j = 0; j < len2; j++) ch2[j] = (Math.random() * 2 - 1) * Math.exp(-j / (len2 * 0.08))
      const src2 = ctx.createBufferSource()
      src2.buffer = buf2
      const f2 = ctx.createBiquadFilter()
      f2.type = 'highpass'
      f2.frequency.value = 7000
      const g2 = ctx.createGain()
      g2.gain.value = 0.5 + (h === 0 ? 0.2 : 0)
      src2.connect(f2)
      f2.connect(g2)
      g2.connect(bgmGain)
      src2.start(ht)
    }
  }

  // ---- 激昂铜管风格旋律 (锯齿波+滤波 = 明亮音色) ----
  // 大调五声: C5=523, D5=587, E5=659, G5=784, A5=880, C6=1047
  const melody = [
    // 第一乐句: 上行爆发
    { note: 523, start: 0, dur: 0.5 },
    { note: 659, start: 0.5, dur: 0.5 },
    { note: 784, start: 1, dur: 1 },
    { note: 880, start: 2, dur: 0.5 },
    { note: 784, start: 2.5, dur: 0.5 },
    { note: 659, start: 3, dur: 0.5 },
    { note: 784, start: 3.5, dur: 0.5 },
    // 第二乐句: 高潮
    { note: 1047, start: 4, dur: 1 },
    { note: 880, start: 5, dur: 0.5 },
    { note: 784, start: 5.5, dur: 0.5 },
    { note: 659, start: 6, dur: 0.5 },
    { note: 784, start: 6.5, dur: 0.5 },
    { note: 880, start: 7, dur: 1 },
    // 第三乐句: 激情推进
    { note: 784, start: 8, dur: 0.5 },
    { note: 880, start: 8.5, dur: 0.5 },
    { note: 1047, start: 9, dur: 0.5 },
    { note: 880, start: 9.5, dur: 0.5 },
    { note: 784, start: 10, dur: 1 },
    { note: 659, start: 11, dur: 0.5 },
    { note: 523, start: 11.5, dur: 0.5 },
    // 第四乐句: 回到主题
    { note: 659, start: 12, dur: 0.5 },
    { note: 784, start: 12.5, dur: 0.5 },
    { note: 880, start: 13, dur: 1 },
    { note: 784, start: 14, dur: 0.5 },
    { note: 659, start: 14.5, dur: 0.5 },
    { note: 523, start: 15, dur: 1 },
  ]

  melody.forEach(m => {
    const mt = t + m.start * beat
    const dur = m.dur * beat

    // 明亮铜管音色 - 锯齿波
    const osc = ctx.createOscillator()
    const g = ctx.createGain()
    const f = ctx.createBiquadFilter()
    osc.type = 'sawtooth'
    osc.frequency.value = m.note
    f.type = 'lowpass'
    f.frequency.value = m.note * 3
    f.Q.value = 1.5
    osc.connect(f)
    f.connect(g)
    g.connect(bgmGain)
    // 快速起音，充满活力
    g.gain.setValueAtTime(0, mt)
    g.gain.linearRampToValueAtTime(0.55, mt + 0.02)
    g.gain.setValueAtTime(0.45, mt + dur * 0.6)
    g.gain.exponentialRampToValueAtTime(0.001, mt + dur)
    osc.start(mt)
    osc.stop(mt + dur)

    // 轻微颤音增加活力
    const vib = ctx.createOscillator()
    const vibG = ctx.createGain()
    vib.type = 'sine'
    vib.frequency.value = 6.5
    vibG.gain.value = 4
    vib.connect(vibG)
    vibG.connect(osc.frequency)
    vib.start(mt)
    vib.stop(mt + dur)
  })

  // ---- 激昂琶音 (赌场风格快速上行音阶) ----
  const arpeggios = [
    { notes: [523, 659, 784, 1047], start: 1.75 },
    { notes: [587, 784, 880, 1175], start: 5.75 },
    { notes: [523, 659, 784, 1047], start: 9.75 },
    { notes: [440, 523, 659, 880], start: 13.75 },
  ]

  arpeggios.forEach(a => {
    a.notes.forEach((note, i) => {
      const at = t + (a.start + i * 0.06) * beat
      const osc = ctx.createOscillator()
      const g = ctx.createGain()
      osc.type = 'triangle'
      osc.frequency.value = note
      osc.connect(g)
      g.connect(bgmGain)
      g.gain.setValueAtTime(0, at)
      g.gain.linearRampToValueAtTime(0.5, at + 0.01)
      g.gain.exponentialRampToValueAtTime(0.001, at + 0.2)
      osc.start(at)
      osc.stop(at + 0.25)
    })
  })

  // ---- 强劲贝斯线 (八度跳跃+节奏推进) ----
  const bass = [
    { note: 131, start: 0, dur: 1 },
    { note: 262, start: 1, dur: 0.5 },
    { note: 131, start: 1.5, dur: 0.5 },
    { note: 147, start: 2, dur: 1 },
    { note: 294, start: 3, dur: 0.5 },
    { note: 147, start: 3.5, dur: 0.5 },
    { note: 165, start: 4, dur: 1 },
    { note: 330, start: 5, dur: 0.5 },
    { note: 165, start: 5.5, dur: 0.5 },
    { note: 147, start: 6, dur: 1 },
    { note: 294, start: 7, dur: 0.5 },
    { note: 147, start: 7.5, dur: 0.5 },
    { note: 131, start: 8, dur: 1 },
    { note: 262, start: 9, dur: 0.5 },
    { note: 131, start: 9.5, dur: 0.5 },
    { note: 165, start: 10, dur: 1 },
    { note: 330, start: 11, dur: 0.5 },
    { note: 165, start: 11.5, dur: 0.5 },
    { note: 147, start: 12, dur: 1 },
    { note: 294, start: 13, dur: 0.5 },
    { note: 147, start: 13.5, dur: 0.5 },
    { note: 131, start: 14, dur: 1 },
    { note: 262, start: 15, dur: 0.5 },
    { note: 131, start: 15.5, dur: 0.5 },
  ]

  bass.forEach(b => {
    const bt = t + b.start * beat
    const dur = b.dur * beat
    const osc = ctx.createOscillator()
    const g = ctx.createGain()
    const f = ctx.createBiquadFilter()
    osc.type = 'sawtooth'
    osc.frequency.value = b.note
    f.type = 'lowpass'
    f.frequency.value = 300
    osc.connect(f)
    f.connect(g)
    g.connect(bgmGain)
    g.gain.setValueAtTime(0, bt)
    g.gain.linearRampToValueAtTime(1.0, bt + 0.02)
    g.gain.exponentialRampToValueAtTime(0.3, bt + 0.08)
    g.gain.exponentialRampToValueAtTime(0.001, bt + dur)
    osc.start(bt)
    osc.stop(bt + dur)
  })

  // ---- 赌场闪光音效装饰 (高频闪烁) ----
  const sparkles = [
    { note: 2093, start: 3.8 },
    { note: 2349, start: 7.8 },
    { note: 2093, start: 11.8 },
    { note: 2637, start: 15.8 },
  ]

  sparkles.forEach(s => {
    const st = t + s.start * beat
    const osc = ctx.createOscillator()
    const g = ctx.createGain()
    osc.type = 'sine'
    osc.frequency.value = s.note
    osc.connect(g)
    g.connect(bgmGain)
    g.gain.setValueAtTime(0, st)
    g.gain.linearRampToValueAtTime(0.25, st + 0.01)
    g.gain.exponentialRampToValueAtTime(0.001, st + 0.12)
    osc.start(st)
    osc.stop(st + 0.15)
  })

  // ~7.7s loop (16 beats * 0.48s)
  const loopMs = 16 * beat * 1000
  bgmTimer = setTimeout(() => scheduleCasinoBGM(ctx), loopMs)
}

export function stopBGM() {
  bgmPlaying = false
  if (bgmTimer) { clearTimeout(bgmTimer); bgmTimer = null }
  if (bgmGain) { bgmGain.gain.value = 0 }
}

export function toggleBGM() {
  if (bgmPlaying) { stopBGM(); return false }
  else { startBGM(); return true }
}

// ============================================================
// CF风格紧张音乐 - 用户操作时播放
// 快节奏电子鼓+紧张合成器+驱动低音
// ============================================================
export function startActionBGM() {
  if (actionPlaying) return
  try {
    const ctx = getCtx()
    actionGain = ctx.createGain()
    actionGain.gain.value = 0.05
    actionGain.connect(ctx.destination)
    actionPlaying = true
    // 降低背景BGM音量
    if (bgmGain && bgmPlaying) bgmGain.gain.value = 0.015
    scheduleActionBGM(ctx)
  } catch (e) {}
}

function scheduleActionBGM(ctx) {
  if (!actionPlaying) return
  const t = ctx.currentTime + 0.05
  const bpm = 140
  const beat = 60 / bpm // ~0.43s

  // ---- 快速电子鼓 ----
  // 4/4 kick on every beat, hi-hat on 8ths, snare on 2&4
  for (let i = 0; i < 16; i++) {
    const bt = t + i * beat

    // Kick on every beat
    if (i % 2 === 0) {
      const osc = ctx.createOscillator()
      const g = ctx.createGain()
      osc.type = 'sine'
      osc.frequency.setValueAtTime(150, bt)
      osc.frequency.exponentialRampToValueAtTime(35, bt + 0.1)
      osc.connect(g)
      g.connect(actionGain)
      g.gain.setValueAtTime(2.0, bt)
      g.gain.exponentialRampToValueAtTime(0.001, bt + 0.15)
      osc.start(bt)
      osc.stop(bt + 0.15)
    }

    // Snare on 2 & 4 (indices 2,6,10,14)
    if (i % 4 === 2) {
      const len = ctx.sampleRate * 0.06
      const buf = ctx.createBuffer(1, len, ctx.sampleRate)
      const ch = buf.getChannelData(0)
      for (let j = 0; j < len; j++) ch[j] = (Math.random() * 2 - 1) * Math.exp(-j / (len * 0.15))
      const src = ctx.createBufferSource()
      src.buffer = buf
      const f = ctx.createBiquadFilter()
      f.type = 'highpass'
      f.frequency.value = 1500
      const g = ctx.createGain()
      g.gain.value = 1.5
      src.connect(f)
      f.connect(g)
      g.connect(actionGain)
      src.start(bt)
    }

    // Hi-hat on every 8th note
    {
      const len = ctx.sampleRate * 0.02
      const buf = ctx.createBuffer(1, len, ctx.sampleRate)
      const ch = buf.getChannelData(0)
      for (let j = 0; j < len; j++) ch[j] = (Math.random() * 2 - 1) * Math.exp(-j / (len * 0.1))
      const src = ctx.createBufferSource()
      src.buffer = buf
      const f = ctx.createBiquadFilter()
      f.type = 'highpass'
      f.frequency.value = 6000
      const g = ctx.createGain()
      g.gain.value = 0.6
      src.connect(f)
      f.connect(g)
      g.connect(actionGain)
      src.start(bt)
    }
  }

  // ---- 紧张合成器旋律 (方波+滤波) ----
  // 小调紧张感: E4-G4-A4-B4 反复
  const actionMelody = [
    { note: 330, start: 0, dur: 1 },
    { note: 392, start: 1, dur: 1 },
    { note: 440, start: 2, dur: 0.5 },
    { note: 494, start: 2.5, dur: 0.5 },
    { note: 440, start: 3, dur: 1 },
    { note: 330, start: 4, dur: 0.5 },
    { note: 370, start: 4.5, dur: 0.5 },
    { note: 440, start: 5, dur: 1 },
    { note: 494, start: 6, dur: 1 },
    { note: 440, start: 7, dur: 0.5 },
    { note: 392, start: 7.5, dur: 0.5 },
    { note: 330, start: 8, dur: 1 },
    { note: 392, start: 9, dur: 1 },
    { note: 494, start: 10, dur: 0.5 },
    { note: 523, start: 10.5, dur: 0.5 },
    { note: 494, start: 11, dur: 1 },
    { note: 440, start: 12, dur: 1 },
    { note: 392, start: 13, dur: 1 },
    { note: 370, start: 14, dur: 1 },
    { note: 330, start: 15, dur: 1 },
  ]

  actionMelody.forEach(m => {
    const mt = t + m.start * beat
    const dur = m.dur * beat
    const osc = ctx.createOscillator()
    const g = ctx.createGain()
    const f = ctx.createBiquadFilter()
    osc.type = 'square'
    osc.frequency.value = m.note
    f.type = 'lowpass'
    f.frequency.value = m.note * 3
    f.Q.value = 4
    osc.connect(f)
    f.connect(g)
    g.connect(actionGain)
    g.gain.setValueAtTime(0, mt)
    g.gain.linearRampToValueAtTime(0.35, mt + 0.02)
    g.gain.setValueAtTime(0.3, mt + dur * 0.6)
    g.gain.exponentialRampToValueAtTime(0.001, mt + dur)
    osc.start(mt)
    osc.stop(mt + dur)
  })

  // ---- 驱动低音 (脉冲式) ----
  const actionBass = [
    { note: 82, start: 0, dur: 2 },
    { note: 98, start: 2, dur: 2 },
    { note: 110, start: 4, dur: 2 },
    { note: 98, start: 6, dur: 2 },
    { note: 82, start: 8, dur: 2 },
    { note: 98, start: 10, dur: 2 },
    { note: 110, start: 12, dur: 2 },
    { note: 82, start: 14, dur: 2 },
  ]

  actionBass.forEach(b => {
    const bt2 = t + b.start * beat
    const dur = b.dur * beat
    const osc = ctx.createOscillator()
    const g = ctx.createGain()
    osc.type = 'sawtooth'
    osc.frequency.value = b.note
    const f = ctx.createBiquadFilter()
    f.type = 'lowpass'
    f.frequency.value = 200
    osc.connect(f)
    f.connect(g)
    g.connect(actionGain)
    g.gain.setValueAtTime(0, bt2)
    g.gain.linearRampToValueAtTime(1.0, bt2 + 0.02)
    g.gain.exponentialRampToValueAtTime(0.3, bt2 + 0.1)
    g.gain.exponentialRampToValueAtTime(0.001, bt2 + dur)
    osc.start(bt2)
    osc.stop(bt2 + dur)
  })

  // ~6.8s loop (16 beats * 0.43s)
  const loopMs = 16 * beat * 1000
  actionTimer = setTimeout(() => scheduleActionBGM(ctx), loopMs)
}

export function stopActionBGM() {
  actionPlaying = false
  if (actionTimer) { clearTimeout(actionTimer); actionTimer = null }
  if (actionGain) { actionGain.gain.value = 0 }
  // 恢复背景BGM音量
  if (bgmGain && bgmPlaying) bgmGain.gain.value = 0.045
}
