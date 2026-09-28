#!/usr/bin/env python3
"""Synthesises The Bakery's sound cues. Standard library only.

Run from this folder: python3 make.py. Every sound is made here from sine
waves; none is recorded or taken from anywhere else (see LICENSES.md).
"""
import math
import struct
import wave

RATE = 22050


def note(freq, start, length, volume=0.5, harmonics=(1.0, 0.3, 0.1)):
    """A plucked note: a few harmonics, a quick attack and a soft decay."""
    samples = {}
    for i in range(int(length * RATE)):
        t = i / RATE
        env = min(1.0, t / 0.008) * math.exp(-t * 6.0)
        v = sum(a * math.sin(2 * math.pi * freq * (k + 1) * t) for k, a in enumerate(harmonics))
        samples[int(start * RATE) + i] = v * env * volume / sum(harmonics)
    return samples


def write(name, *notes):
    mix = {}
    for n in notes:
        for i, v in n.items():
            mix[i] = mix.get(i, 0.0) + v
    frames = bytearray()
    for i in range(max(mix) + 1):
        v = max(-1.0, min(1.0, mix.get(i, 0.0)))
        frames += struct.pack('<h', int(v * 32000))
    with wave.open(name, 'wb') as w:
        w.setnchannels(1)
        w.setsampwidth(2)
        w.setframerate(RATE)
        w.writeframes(bytes(frames))


C5, E5, G5, C6, G4, B4, A4, E4, D4 = 523.25, 659.25, 783.99, 1046.5, 392.0, 493.88, 440.0, 329.63, 293.66

# A task reaches Done: two rising notes.
write('task-done.wav', note(C5, 0, 0.3), note(G5, 0.09, 0.35))
# A letter arrives: a three-note chime.
write('letter.wav', note(E5, 0, 0.35, 0.4), note(G5, 0.1, 0.35, 0.4), note(C6, 0.2, 0.5, 0.4))
# A run finished: a warm chord.
write('run-finished.wav', note(G4, 0, 0.5, 0.45), note(B4, 0.02, 0.5, 0.4), note(D4 * 2, 0.04, 0.5, 0.3))
# A run failed: two falling notes, duller.
write('run-failed.wav', note(A4, 0, 0.3, 0.5, (1.0, 0.5, 0.35)), note(E4, 0.16, 0.45, 0.5, (1.0, 0.5, 0.35)))
