// Simple end-to-end tests for Viction JSON-RPC APIs.
// Usage: RPC_URL=http://localhost:8547 npm test

const RPC_URL = process.env.RPC_URL ?? "http://localhost:8547";
const EPOCH = Number(process.env.EPOCH ?? 900);

// ---------------------------------------------------------------------------
// Minimal JSON-RPC client
// ---------------------------------------------------------------------------

async function call<T = unknown>(method: string, params: unknown[] = []): Promise<T> {
  const res = await fetch(RPC_URL, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ jsonrpc: "2.0", id: 1, method, params }),
  });
  const json = await res.json() as { result?: T; error?: { code: number; message: string } };
  if (json.error) throw new Error(`[${json.error.code}] ${json.error.message}`);
  return json.result as T;
}

// ---------------------------------------------------------------------------
// Test runner
// ---------------------------------------------------------------------------

let passed = 0;
let failed = 0;

async function test(name: string, fn: () => Promise<unknown>): Promise<void> {
  try {
    const note = await fn();
    console.log(`  ✔ ${name}${note ? ` — ${note}` : ""}`);
    passed++;
  } catch (e) {
    console.log(`  ✘ ${name}`);
    console.log(`    ${(e as Error).message}`);
    failed++;
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function isAddress(s: unknown): boolean {
  return typeof s === "string" && /^0x[0-9a-fA-F]{40}$/.test(s);
}
function isHash(s: unknown): boolean {
  return typeof s === "string" && /^0x[0-9a-fA-F]{64}$/.test(s);
}
function isHex(s: unknown): boolean {
  return typeof s === "string" && /^0x[0-9a-fA-F]*$/.test(s);
}


// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

async function main(): Promise<void> {
  console.log(`endpoint: ${RPC_URL}\n`);

  // Discover useful fixtures from the chain.
  const latest = await call<{ number: string; hash: string }>("eth_getBlockByNumber", ["latest", false]);
  if (!latest) throw new Error("node returned null for latest block — still syncing?");
  const headNum = parseInt(latest.number, 16);
  const headHash = latest.hash;

  // Latest checkpoint block (2*epoch or above, at least 1 epoch below head).
  let ckptHash: string | null = null;
  if (headNum >= 2 * EPOCH) {
    const ckptNum = Math.floor((headNum - EPOCH) / EPOCH) * EPOCH;
    if (ckptNum >= 2 * EPOCH) {
      const blk = await call<{ hash: string } | null>("eth_getBlockByNumber", ["0x" + ckptNum.toString(16), false]);
      if (blk) ckptHash = blk.hash;
    }
  }

  console.log(`head:  #${headNum} (${headHash})`);
  console.log(`ckpt:  ${ckptHash ? `${ckptHash}` : "(none – chain below 2*epoch)"}\n`);

  // ── eth_getHeaderByNumber ──────────────────────────────────────────────
  console.log("eth_getHeaderByNumber");
  await test("'latest' returns a well-formed header", async () => {
    const h = await call<Record<string, unknown>>("eth_getHeaderByNumber", ["latest"]);
    if (!h) throw new Error("null result");
    if (!isHash(h.hash)) throw new Error(`bad hash: ${h.hash}`);
    if (!isHex(h.number)) throw new Error(`bad number: ${h.number}`);
    if (!isAddress(h.miner)) throw new Error(`bad miner: ${h.miner}`);
    return `#${parseInt(h.number as string, 16)}`;
  });

  await test("specific block number returns correct number", async () => {
    const target = Math.max(1, headNum - 5)
    const h = await call<Record<string, unknown>>("eth_getHeaderByNumber", ["0x" + target.toString(16)]);
    if (!h) throw new Error("null result");
    if (parseInt(h.number as string, 16) !== target) throw new Error(`expected #${target} got #${parseInt(h.number as string, 16)}`);
    return `#${target}`;
  });

  await test("specific block number returns at checkpoint number", async () => {
    const ckptNum = Math.floor((headNum - EPOCH) / EPOCH) * EPOCH;
    const h = await call<Record<string, unknown>>("eth_getHeaderByNumber", ["0x" + ckptNum.toString(16)]);
    if (!h) throw new Error("null result");
    if (parseInt(h.number as string, 16) !== ckptNum) throw new Error(`expected #${ckptNum} got #${parseInt(h.number as string, 16)}`);
    console.log("header ", h);
    return `#${ckptNum}`;
  });

  await test("unknown future block returns null", async () => {
    const h = await call<unknown>("eth_getHeaderByNumber", ["0x" + (headNum + 1_000_000).toString(16)]);
    if (h !== null) throw new Error(`expected null, got ${JSON.stringify(h)}`);
  });

  // ── eth_getHeaderByHash ───────────────────────────────────────────────
  console.log("\neth_getHeaderByHash");
  await test("known hash returns correct header", async () => {
    const h = await call<Record<string, unknown>>("eth_getHeaderByHash", [headHash]);
    if (!h) throw new Error("null result");
    if ((h.hash as string).toLowerCase() !== headHash.toLowerCase()) throw new Error("hash mismatch");
    if (!isAddress(h.miner)) throw new Error(`bad miner: ${h.miner}`);
    return `#${parseInt(h.number as string, 16)}`;
  });

  await test("unknown hash returns null", async () => {
    const h = await call<unknown>("eth_getHeaderByHash", ["0x" + "aa".repeat(32)]);
    if (h !== null) throw new Error(`expected null, got ${JSON.stringify(h)}`);
  });

  if (ckptHash) {
    await test("checkpoint block has posv=true and validators[]", async () => {
      const h = await call<Record<string, unknown>>("eth_getHeaderByHash", [ckptHash]);
      if (!h) throw new Error("null result");
      if (h.posv !== true) throw new Error("posv field must be true");
      const validators = h.validators as string[] | undefined;
      if (!Array.isArray(validators) || validators.length === 0) throw new Error("validators[] missing or empty");
      if (!validators.every(isAddress)) throw new Error("invalid address in validators[]");
      return `validators=${validators.length}`;
    });
  }

  // ── eth_getAttestorsPairsByNumber ──────────────────────────────────────
  console.log("\neth_getAttestorsPairsByNumber");
  await test("'latest' returns creator→attestor map", async () => {
    const pairs = await call<Record<string, string>>("eth_getAttestorsPairsByNumber", ["latest"]);
    if (!pairs || typeof pairs !== "object") throw new Error("not an object");
    for (const [k, v] of Object.entries(pairs)) {
      if (!isAddress(k)) throw new Error(`bad creator: ${k}`);
      if (!isAddress(v)) throw new Error(`bad attestor: ${v}`);
    }
    console.log("pairs ", pairs);
    return `${Object.keys(pairs).length} pairs`;
  });

  await test("unknown future block returns error", async () => {
    await expectError(
      () => call("eth_getAttestorsPairsByNumber", ["0x" + (headNum + 1_000_000).toString(16)]),
      ["header not found", "not found"],
    );
  });

  // ── eth_getAttestorsPairsByHash ────────────────────────────────────────
  console.log("\neth_getAttestorsPairsByHash");
  await test("head hash returns same pairs as byNumber('latest')", async () => {
    const byNum = await call<Record<string, string>>("eth_getAttestorsPairsByNumber", ["latest"]);
    const byHash = await call<Record<string, string>>("eth_getAttestorsPairsByHash", [headHash]);
    const a = JSON.stringify(Object.fromEntries(Object.entries(byNum).sort()));
    const b = JSON.stringify(Object.fromEntries(Object.entries(byHash).sort()));
    if (a !== b) throw new Error(`byNumber vs byHash mismatch`);
  });

  await test("unknown hash returns error", async () => {
    await expectError(
      () => call("eth_getAttestorsPairsByHash", ["0x" + "cd".repeat(32)]),
      ["header not found", "not found"],
    );
  });

  // ── eth_getRewardByHash ────────────────────────────────────────────────
  console.log("\neth_getRewardByHash");
  if (ckptHash) {
    await test("checkpoint hash returns signers and rewards", async () => {
      const r = await call<{ signers: Record<string, unknown>; rewards: Record<string, string> }>(
        "eth_getRewardByHash", [ckptHash],
      );
      if (!r?.signers) throw new Error("missing signers");
      if (!r?.rewards) throw new Error("missing rewards");
      for (const addr of Object.keys(r.signers)) {
        if (!isAddress(addr)) throw new Error(`bad signer address: ${addr}`);
      }
      console.log("reward ", r);
      return `signers=${Object.keys(r.signers).length} stakeholders=${Object.keys(r.rewards).length}`;
    });
  } else {
    console.log("  ○ checkpoint hash returns signers and rewards (skip: no checkpoint)");
  }

  await test("non-checkpoint hash returns error", async () => {
    await expectError(
      () => call("eth_getRewardByHash", [headHash]),
      ["not a checkpoint", "header is not a checkpoint"],
      { allowPass: headNum % EPOCH === 0 },  // head itself might be a checkpoint
    );
  });

  await test("unknown hash returns error", async () => {
    await expectError(
      () => call("eth_getRewardByHash", ["0x" + "ef".repeat(32)]),
      ["not a checkpoint", "header is not a checkpoint"],
    );
  });

  // ── eth_getBlockFinalityByNumber ───────────────────────────────────────
  console.log("\neth_getBlockFinalityByNumber");
  await test("genesis block returns 100", async () => {
    const f = await call<number>("eth_getBlockFinalityByNumber", ["0x0"]);
    if (f !== 100) throw new Error(`expected 100, got ${f}`);
  });

  await test("recent block returns finality in [0, 100]", async () => {
    const target = Math.max(1, headNum - 50);
    const f = await call<number>("eth_getBlockFinalityByNumber", ["0x" + target.toString(16)]);
    if (typeof f !== "number" || f < 0 || f > 100) throw new Error(`out of range: ${f}`);
    return `block #${target} = ${f}%`;
  });

  await test("unknown future block returns 0", async () => {
    const f = await call<number>("eth_getBlockFinalityByNumber", ["0x" + (headNum + 1_000_000).toString(16)]);
    if (f !== 0) throw new Error(`expected 0, got ${f}`);
  });

  // ── eth_getBlockFinalityByHash ─────────────────────────────────────────
  console.log("\neth_getBlockFinalityByHash");
  await test("head hash returns finality in [0, 100]", async () => {
    const f = await call<number>("eth_getBlockFinalityByHash", [headHash]);
    if (typeof f !== "number" || f < 0 || f > 100) throw new Error(`out of range: ${f}`);
    return `${f}%`;
  });

  await test("unknown hash returns 0", async () => {
    const f = await call<number>("eth_getBlockFinalityByHash", ["0x" + "ab".repeat(32)]);
    if (f !== 0) throw new Error(`expected 0, got ${f}`);
  });

  // ── Summary ────────────────────────────────────────────────────────────
  console.log(`\n${failed === 0 ? "✓" : "✗"} ${passed} passed, ${failed} failed`);
  if (failed > 0) process.exit(1);
}

// Asserts fn() throws an error whose message contains one of the expected strings.
// allowPass: if true, a successful call is also accepted (e.g. when the block
// might legitimately not be an error).
async function expectError(
  fn: () => Promise<unknown>,
  expected: string[],
  opts: { allowPass?: boolean } = {},
): Promise<void> {
  try {
    await fn();
    if (opts.allowPass) return;
    throw new Error(`expected error containing one of: ${expected.join(", ")}`);
  } catch (e) {
    if (e instanceof Error && e.message.startsWith("expected error")) throw e;
    const msg = (e as Error).message.toLowerCase();
    if (!expected.some(s => msg.includes(s.toLowerCase()))) {
      throw new Error(`error "${(e as Error).message}" did not match: ${expected.join(", ")}`);
    }
  }
}

main().catch(e => {
  console.error("Fatal:", (e as Error).message);
  process.exit(1);
});
