# Regenerate with: python3 bcvectors.py . (requires Pillow)
"""Generate BCn reference vectors: random blocks decoded by Pillow's DDS plugin."""
import io, os, random, struct, sys
from PIL import Image

out = sys.argv[1]
random.seed(20261009)

def dds(fourcc, w, h, data):
    pf = struct.pack('<II4sIIIII', 32, 0x4, fourcc, 0, 0, 0, 0, 0)
    hdr = struct.pack('<IIIIIII', 124, 0x1 | 0x2 | 0x4 | 0x1000 | 0x80000, h, w, len(data), 0, 0)
    hdr += b'\0' * 44 + pf + struct.pack('<IIIII', 0x1000, 0, 0, 0, 0)
    return b'DDS ' + hdr + data

formats = {'DXT1': (b'DXT1', 8), 'DXT3': (b'DXT3', 16), 'DXT5': (b'DXT5', 16), 'ATI1N': (b'ATI1', 8), 'ATI2N': (b'ATI2', 16)}
for name, (fourcc, block) in formats.items():
    for w, h in [(16, 16), (6, 10)]:
        blocks = ((w + 3) // 4) * ((h + 3) // 4)
        data = bytearray(random.getrandbits(8) for _ in range(blocks * block))
        # Force both endpoint orderings to appear in the first two blocks
        if name in ('DXT1',):
            data[0:4] = b'\x00\xf8\x1f\x00'  # c0 > c1: 4 colour mode
            data[8:12] = b'\x1f\x00\x00\xf8'  # c0 < c1: 3 colour mode
        im = Image.open(io.BytesIO(dds(fourcc, w, h, bytes(data))))
        im.load()
        base = os.path.join(out, f'{name}_{w}x{h}')
        open(base + '.bin', 'wb').write(data)
        open(base + '.' + im.mode.lower(), 'wb').write(im.tobytes())
        print(base, im.mode, im.size)
