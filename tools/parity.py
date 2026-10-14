"""parity.py [OUTDIR]: run the original MAIN.EXE (LOLARGS, default 3: a new game with Conrad; PLAIN: the menu) in DOSBox-X (remote-debug fork, host input
disabled), drive it with the keys TestNewGame uses, and dump the mode-13h screen (palette indices)
as OUTDIR/<step>.raw for comparing with the Go port's dumps (LOLSHOT). Game files come from
../game, MAIN.EXE from lands-of-lore-1-re/decomp/orig."""
import os, sys, time, socket, json, subprocess, tempfile

R = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
out = sys.argv[1] if len(sys.argv) > 1 else tempfile.mkdtemp(prefix='lolparity')
os.makedirs(out, exist_ok=True)
for f in os.listdir(out):
    if f.endswith('.raw'):
        os.remove(os.path.join(out, f))

class Qmp:
    def __init__(self, port=4444):
        self.s = socket.create_connection(('127.0.0.1', port), timeout=60)
        self.f = self.s.makefile('rwb')
        self._read()
        self.cmd('qmp_capabilities')
    def _read(self):
        while True:
            o = json.loads(self.f.readline())
            if 'event' not in o:
                return o
    def cmd(self, name, args=None):
        o = {'execute': name}
        if args:
            o['arguments'] = args
        self.f.write((json.dumps(o) + '\n').encode()); self.f.flush()
        r = self._read()
        if 'error' in r:
            raise RuntimeError(r['error'])
        return r.get('return')
    def key(self, k):
        # press and release apart: send-key queues both into one tick, which the game misses
        for down in (True, False):
            self.cmd('input-send-event', {'events': [{'type': 'key', 'data': {'down': down, 'key': {'type': 'qcode', 'data': k}}}]})
            time.sleep(0.15)
    def screen(self, name):
        f = os.path.join(out, name + '.bin')
        self.cmd('memdump', {'address': 0xa0000, 'size': 64000, 'file': f}); time.sleep(0.5)
        os.replace(f, os.path.join(out, name + '.raw'))

game = tempfile.mkdtemp(prefix='lolgame')
src = os.path.join(R, 'game')
for n in os.listdir(src):
    os.symlink(os.path.join(src, n), os.path.join(game, n))
exe = os.path.expanduser('~/Projects/lands-of-lore-1-re/decomp/orig/MAIN.EXE')
subprocess.run(['cp', exe, game + '/MAIN.EXE'], check=True)
ARGS = os.environ.get('LOLARGS', '3')  # PLAIN shows the menu; a number starts that hero's new game
# LANDS CHARGEN, LANDS INTRO, LANDS HISTORY run the language folder's LANDS.EXE instead and dump
# the screen every LOLEVERY seconds (default 2) for LOLFOR seconds (default 30)
LANDS = ARGS.upper().startswith('LANDS')
run = 'main.exe %s' % ARGS
if LANDS:
    eng = os.path.join(game, 'ENG')
    os.remove(eng)
    os.mkdir(eng)
    for n in os.listdir(os.path.join(src, 'ENG')):
        os.symlink(os.path.join(src, 'ENG', n), os.path.join(eng, n))
    subprocess.run(['cp', os.path.expanduser('~/Projects/lands-of-lore-1-re/decomp/lands/LANDS.EXE'), eng], check=True)
    run = 'cd ENG\n' + ARGS.lower().replace('lands', 'lands.exe c:\\\\', 1)  # argv[1] is the game's folder
conf = os.path.join(game, 'parity.conf')
open(conf, 'w').write('[sdl]\noutput=surface\nautolock=false\n[dosbox]\nmachine=svga_s3\nmemsize=16\nqmpserver=true\n'
                      '[cpu]\ncore=normal\ncycles=fixed 30000\n[sblaster]\nsbtype=sb16\n'
                      '[autoexec]\nmount c "%s"\nc:\n%s\n' % (game, run))
p = subprocess.Popen([os.path.expanduser('~/tools/dosbox-x-remotedebug/src/dosbox-x'), '-conf', conf],
                     env=dict(os.environ, DOSBOX_NO_HOST_INPUT='1'), stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
try:
    for _ in range(150):
        try:
            q = Qmp(); break
        except OSError:
            time.sleep(0.2)
    else:
        sys.exit('no QMP connection to DOSBox-X')
    if LANDS:
        every = float(os.environ.get('LOLEVERY', '2'))
        for i in range(int(float(os.environ.get('LOLFOR', '30')) / every)):
            time.sleep(every)
            q.screen('l%03d' % i)
        f = os.path.join(out, 'text.bin')
        q.cmd('memdump', {'address': 0xb8000, 'size': 4000, 'file': f}); time.sleep(0.5)
        print(open(f, 'rb').read()[::2].decode('latin1'))
        sys.exit(0)
    time.sleep(12)
    if ARGS.upper() == 'PLAIN':
        q.screen('menu')
        sys.exit(0)
    for i in range(12):
        time.sleep(4)
        if os.environ.get('TRACE'):
            q.screen('t%02d' % i)
        q.key('ret')
    time.sleep(3); q.screen('game')
    for k, name in [('left', 'strafe'), ('up', 'bump'), ('right', 'strafe2'), ('home', 'turn'), ('up', 'walk'), ('down', 'back')]:
        q.key(k); time.sleep(float(os.environ.get('LOLSTEPWAIT', '3'))); q.screen(name)
finally:
    p.kill()
print(out)
