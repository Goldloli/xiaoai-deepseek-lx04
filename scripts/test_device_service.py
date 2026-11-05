#!/usr/bin/env python3
"""Exercise service recovery with fake Android commands and no hardware access."""
# SPDX-License-Identifier: Apache-2.0
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import time
import unittest

ROOT = Path(__file__).resolve().parents[1]


class DeviceServiceTest(unittest.TestCase):
    def test_late_network_failed_start_duplicate_launch_and_shutdown(self):
        with tempfile.TemporaryDirectory(prefix='lx04-service-') as temporary:
            root = Path(temporary)
            app, commands = root / 'app', root / 'bin'
            app.mkdir()
            commands.mkdir()
            (app / 'config.json').write_text('{}')
            (app / 'dns-server.txt').write_text('127.0.0.1:53\n')

            def executable(path, source):
                path.write_text(source)
                path.chmod(0o700)

            executable(commands / 'getprop', '#!/bin/sh\necho 1\n')
            executable(commands / 'ip', f'''#!{sys.executable}
import pathlib, sys
root = pathlib.Path({str(app)!r})
if sys.argv[1:] != ['-4', 'route', 'get', '1.1.1.1']:
    sys.exit(2)
path = root / 'network-checks'
count = int(path.read_text()) + 1 if path.exists() else 1
path.write_text(str(count))
sys.exit(1 if count == 1 else 0)
''')
            # macOS has flock(2), but does not ship the flock command.
            executable(commands / 'flock', f'''#!{sys.executable}
import fcntl, sys
try:
    fcntl.flock(int(sys.argv[-1]), fcntl.LOCK_EX | fcntl.LOCK_NB)
except BlockingIOError:
    sys.exit(1)
''')
            executable(app / 'lx04-agent', f'''#!{sys.executable}
import os, pathlib, signal, sys, time
root = pathlib.Path(__file__).resolve().parent
if '-restore-audio' in sys.argv:
    with (root / 'restores').open('a') as f:
        f.write('restore\\n')
    sys.exit(0)
path = root / 'starts'
count = int(path.read_text()) + 1 if path.exists() else 1
path.write_text(str(count))
if count == 1:
    print('simulated startup network failure', flush=True)
    sys.exit(1)
print('ready', flush=True)
signal.signal(signal.SIGTERM, lambda *_: sys.exit(0))
while True:
    time.sleep(0.05)
''')
            source = (ROOT / 'scripts/device-run.sh').read_text()
            source = source.replace('APP=/data/adb/lx04-deepseek', f'APP={app}')
            source = source.replace('/system/bin/', f'{commands}/')
            launcher = root / 'run.sh'
            launcher.write_text(source)
            env = dict(os.environ, PATH=f'{commands}:{os.environ["PATH"]}')
            env.pop('XIAOBOT_DNS_SERVER', None)
            process = subprocess.Popen(['sh', str(launcher)], env=env,
                                       stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
            child_pid = None
            try:
                deadline = time.monotonic() + 15
                while time.monotonic() < deadline:
                    starts = app / 'starts'
                    ready = app / 'runtime.log'
                    if (starts.exists() and starts.read_text() == '2' and
                            ready.exists() and 'ready' in ready.read_text()):
                        break
                    if process.poll() is not None:
                        self.fail(f'supervisor exited early: {process.stderr.read().decode()}')
                    time.sleep(0.05)
                else:
                    self.fail('service did not recover after late network and failed startup')
                self.assertGreaterEqual(int((app / 'network-checks').read_text()), 2)
                self.assertIn('simulated startup network failure',
                              (app / 'runtime.previous.log').read_text())
                child_pid = int((app / 'agent.pid').read_text())
                duplicate = subprocess.run(['sh', str(launcher)], env=env,
                                           capture_output=True, timeout=3)
                self.assertEqual(duplicate.returncode, 0, duplicate.stderr)
                self.assertEqual((app / 'starts').read_text(), '2')
                process.send_signal(signal.SIGTERM)
                self.assertEqual(process.wait(timeout=5), 0)
                self.assertFalse((app / 'agent.pid').exists())
                self.assertFalse((app / 'supervisor.pid').exists())
                with self.assertRaises(ProcessLookupError):
                    os.kill(child_pid, 0)
                self.assertGreaterEqual(len((app / 'restores').read_text().splitlines()), 2)
            finally:
                if process.poll() is None:
                    process.terminate()
                    try:
                        process.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        process.kill()
                        process.wait()
                if child_pid is not None:
                    try:
                        os.kill(child_pid, signal.SIGTERM)
                    except ProcessLookupError:
                        pass
                process.stderr.close()


if __name__ == '__main__':
    unittest.main()
