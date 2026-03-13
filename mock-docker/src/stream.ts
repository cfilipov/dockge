// Docker multiplexed stream protocol: 8-byte header + payload
// Header: [stream_type (1 byte), 0, 0, 0, payload_size (4 bytes big-endian)]

export function frameOutput(line: string, stream: number = 1): Buffer {
    const payload = Buffer.from(line + "\n", "utf-8");
    const header = Buffer.alloc(8);
    header[0] = stream; // 1 = stdout, 2 = stderr
    header.writeUInt32BE(payload.length, 4);
    return Buffer.concat([header, payload]);
}
