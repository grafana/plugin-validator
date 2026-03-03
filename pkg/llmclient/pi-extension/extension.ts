/**
 * Pi extension for the plugin-validator AgenticClient.
 *
 * Registers:
 * - submit_answer: structured answer tool matching AnswerSchema
 * - bash (override): restricted shell that only allows whitelisted commands
 */

import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { Type } from "@sinclair/typebox";
import { execSync } from "child_process";

// =============================================================================
// submit_answer tool
// =============================================================================

const submitAnswerSchema = Type.Object({
  answer: Type.String({
    description: "Brief answer explaining your findings",
  }),
  short_answer: Type.Boolean({
    description:
      "The direct yes/no answer to the question. Set to true if the answer to the question is YES, or false if the answer is NO. For example, if asked 'Is the sky blue?' set this to true.",
  }),
  files: Type.Optional(
    Type.Array(Type.String(), {
      description: "List of relevant file paths",
    }),
  ),
  code_snippet: Type.Optional(
    Type.String({
      description: "A relevant code snippet illustrating your findings",
    }),
  ),
});

// =============================================================================
// Restricted bash tool
// =============================================================================

const bashSchema = Type.Object({
  command: Type.String({ description: "The shell command to execute" }),
  timeout: Type.Optional(
    Type.Number({ description: "Timeout in seconds (default: 30)" }),
  ),
});

// Commands that are allowed to run (first token of the command).
const ALLOWED_COMMANDS = new Set([
  "git",
  "ls",
  "cat",
  "grep",
  "rg",
  "head",
  "tail",
  "wc",
  "diff",
  "tree",
  "file",
  "stat",
  "sort",
  "uniq",
  "jq",
  "find",
]);

// Git subcommands that are allowed.
// Note: Destructive commands like 'reset' and 'clean' are explicitly permitted
// because agents operate in sandboxed/temporary environments where data loss is acceptable.
const ALLOWED_GIT_SUBCOMMANDS = new Set([
  "log",
  "show",
  "diff",
  "status",
  "ls-files",
  "blame",
  "rev-parse",
  "cat-file",
  "checkout",
  "fetch",
  "pull",
  "branch",
  "tag",
  "remote",
  "shortlog",
  "reset",
  "clean",
]);

// Git flags that could execute arbitrary commands.
const BLOCKED_GIT_FLAGS = [
  "--exec",
  "--ext-diff",
  "--upload-pack",
  "--receive-pack",
  "-c",
  "--config",
  "--hook",
  "--run",
];

/**
 * Parse a command string into tokens, respecting quotes.
 * Handles single quotes, double quotes, and backslash escapes.
 */
function parseCommand(cmd: string): string[] {
  const tokens: string[] = [];
  let current = "";
  let inSingle = false;
  let inDouble = false;
  let escaped = false;

  for (const ch of cmd) {
    if (escaped) {
      current += ch;
      escaped = false;
      continue;
    }
    if (ch === "\\") {
      escaped = true;
      continue;
    }
    if (ch === "'" && !inDouble) {
      inSingle = !inSingle;
      continue;
    }
    if (ch === '"' && !inSingle) {
      inDouble = !inDouble;
      continue;
    }
    if ((ch === " " || ch === "\t") && !inSingle && !inDouble) {
      if (current.length > 0) {
        tokens.push(current);
        current = "";
      }
      continue;
    }
    current += ch;
  }
  if (current.length > 0) {
    tokens.push(current);
  }
  return tokens;
}

/**
 * Validate that a command is allowed. Returns an error message or null.
 */
function validateCommand(tokens: string[]): string | null {
  if (tokens.length === 0) {
    return "Empty command";
  }

  // Handle pipes: validate each segment independently.
  const segments: string[][] = [];
  let current: string[] = [];
  for (const token of tokens) {
    if (token === "|") {
      if (current.length > 0) {
        segments.push(current);
        current = [];
      }
    } else {
      current.push(token);
    }
  }
  if (current.length > 0) {
    segments.push(current);
  }

  for (const segment of segments) {
    const err = validateSingleCommand(segment);
    if (err) return err;
  }
  return null;
}

function validateSingleCommand(tokens: string[]): string | null {
  if (tokens.length === 0) return "Empty command segment";

  const cmd = tokens[0];
  if (!ALLOWED_COMMANDS.has(cmd)) {
    return `Command '${cmd}' is not allowed. Allowed: ${[...ALLOWED_COMMANDS].join(", ")}`;
  }

  // Extra validation for git
  if (cmd === "git") {
    // Find the subcommand (skip flags like --no-pager)
    let subcommand: string | undefined;
    for (let i = 1; i < tokens.length; i++) {
      if (!tokens[i].startsWith("-")) {
        subcommand = tokens[i];
        break;
      }
    }
    if (!subcommand) {
      return "git command requires a subcommand";
    }
    if (!ALLOWED_GIT_SUBCOMMANDS.has(subcommand)) {
      return `git subcommand '${subcommand}' is not allowed. Allowed: ${[...ALLOWED_GIT_SUBCOMMANDS].join(", ")}`;
    }

    // Check for blocked flags
    for (const arg of tokens.slice(1)) {
      for (const blocked of BLOCKED_GIT_FLAGS) {
        const exactOrLongValue =
          arg === blocked || arg.startsWith(blocked + "=");
        // Short flags like -c can have concatenated values: -ccore.pager=evil
        const shortConcatenated =
          blocked.length === 2 &&
          blocked[0] === "-" &&
          blocked[1] !== "-" &&
          arg.startsWith(blocked) &&
          arg.length > blocked.length;
        if (exactOrLongValue || shortConcatenated) {
          return `git flag '${arg}' is not allowed for security reasons`;
        }
      }
    }
  }

  return null;
}

// =============================================================================
// Extension entry point
// =============================================================================

export default function (pi: ExtensionAPI) {
  function ensureGrepToolEnabled() {
    const activeTools = new Set(pi.getActiveTools());
    if (activeTools.has("grep")) {
      return;
    }
    activeTools.add("grep");
    pi.setActiveTools(Array.from(activeTools));
  }

  // Register submit_answer
  pi.registerTool({
    name: "submit_answer",
    label: "Submit Answer",
    description:
      "Submit your final structured answer. You MUST call this tool to provide your answer. For multiple questions, call once per question.",
    parameters: submitAnswerSchema,
    async execute(_toolCallId, params) {
      return {
        content: [
          {
            type: "text" as const,
            text: "Answer recorded successfully. If you have answered all questions, you are done. Otherwise, continue with the next question.",
          },
        ],
        details: params,
      };
    },
  });

  // Override bash with restricted version
  pi.registerTool({
    name: "bash",
    label: "bash (restricted)",
    description:
      "Execute a shell command. Only whitelisted commands are allowed: git (with allowed subcommands), ls, cat, grep, rg, head, tail, wc, diff, tree, file, stat, sort, uniq, jq, find. Pipes between allowed commands are supported.",
    parameters: bashSchema,
    async execute(_toolCallId, params, _signal, _onUpdate, ctx) {
      const tokens = parseCommand(params.command);
      const error = validateCommand(tokens);
      if (error) {
        return {
          content: [{ type: "text" as const, text: `Error: ${error}` }],
          details: { blocked: true },
        };
      }

      const timeoutMs = (params.timeout ?? 30) * 1000;

      try {
        const output = execSync(params.command, {
          cwd: ctx.cwd,
          timeout: timeoutMs,
          maxBuffer: 1024 * 1024, // 1MB
          encoding: "utf-8",
          stdio: ["pipe", "pipe", "pipe"],
        });
        return {
          content: [{ type: "text" as const, text: output || "(no output)" }],
          details: {},
        };
      } catch (err: any) {
        // execSync throws on non-zero exit codes
        const stderr = err.stderr || "";
        const stdout = err.stdout || "";
        const combined = [stdout, stderr].filter(Boolean).join("\n");
        return {
          content: [
            {
              type: "text" as const,
              text: combined || `Command failed with exit code ${err.status}`,
            },
          ],
          details: { exitCode: err.status },
        };
      }
    },
  });

  pi.on("session_start", async () => {
    ensureGrepToolEnabled();
  });
}
