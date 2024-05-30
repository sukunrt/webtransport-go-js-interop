import { server } from './server.js'

const connectionDetails = await server()

console.info(`
Paste the following code into https://codepen.io/pen/?editors=0012 or similar:

(async function main ()  {
  console.info('CLIENT create session')
  const transport = new WebTransport('${connectionDetails.address}', {
    serverCertificateHashes: [${connectionDetails.serverCertificateHashes.map(cert => `{
      algorithm: '${cert.algorithm}',
      value: Uint8Array.from(atob('${btoa(String.fromCodePoint(...cert.value))}'), (m) => m.codePointAt(0))
    }`)}]
  })

  console.info('CLIENT wait for session')
  await transport.ready
  console.info('CLIENT session ready')

  console.info('CLIENT create bidi stream')
  const stream = await transport.createBidirectionalStream()
  const reader = stream.readable.getReader()

  let bytes = 0

  try {
    while (true) {
      const res = await reader.read()

      if (res.done) {
        console.info('CLIENT read stream finished')
        break
      }

      console.info('bytes', bytes)
      bytes += res.value.byteLength
    }

    console.info('CLIENT received', bytes, 'bytes of 268435456')
  } catch (err) {
      console.info('CLIENT read errored', err)
  }
})()
`)

// keep process running
setInterval(() => {}, 1000)
