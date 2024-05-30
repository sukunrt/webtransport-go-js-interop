import { spawn } from 'node:child_process'

export async function server () {
  let resolved = false

  return new Promise((resolve, reject) => {
    console.info('SERVER start')

    const server = spawn('go', ['run', 'server.go'], {
      // detached runs in separate process group
      detached: true
    })

    // stop server process and it's children on process exit
    function killServer () {
      // negative pid kills whole process group
      process.kill(-server.pid, 'SIGKILL')
    }

    // stop server on ctrl+c
    process.on('SIGINT', () => {
      killServer()
      process.exit(0)
    })

    // stop server on error
    process.on('uncaughtException', (err) => {
      killServer()

      console.error(err)
      process.exit(1)
    })

    server.addListener('exit', (code) => {
      if (code !== 0) {
        throw new Error('Server exited with code ' + code)
      }

      console.info('server exited with code', code)
    })
    server.stderr.addListener('data', buf => {
      if (!resolved) {
        reject(new Error(buf.toString()))
      } else {
        console.error(buf.toString())
      }
    })

    server.stdout.addListener('data', buf => {
      if (!resolved) {
        console.info('SERVER ready')
        resolved = true
        const certHash = Buffer.from(buf.toString(), 'base64')

        resolve({
          address: `https://127.0.0.1:12345/say-hello`,
          serverCertificateHashes: [{
            algorithm: 'sha-256',
            value: certHash
          }]
        })
      } else {
        console.info(buf.toString())
      }
    })
  })
}
