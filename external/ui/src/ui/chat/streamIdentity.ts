import type { TranscriptItem } from "./types";

export function countTurnRows(items: readonly TranscriptItem[]): number {
  let n = 0;
  for (const item of items) {
    if (
      item.type === "user_message" ||
      item.type === "background_wake" ||
      item.type === "goal_turn"
    ) {
      n++;
    }
  }
  return n;
}

/** The stable row counters for the turn a new send opens. */
export function newTurnIdentity(
  items: readonly TranscriptItem[],
  turnsBefore: number,
): { turnIdx: number; assistantStep: number; thinkingStep: number } {
  return {
    turnIdx: turnsBefore + countTurnRows(items) + 1,
    assistantStep: 0,
    thinkingStep: 0,
  };
}

/** The stable row counters a reader rejoining a turn already on screen needs. */
export function resumedTurnIdentity(
  items: readonly TranscriptItem[],
  turnsBefore: number,
): {
  turnIdx: number;
  assistantStep: number;
  thinkingStep: number;
  assistantRowUsed: boolean;
} {
  const opened = countTurnRows(items);
  const turnIdx = turnsBefore + opened;
  const assistant = new RegExp(`^as_${turnIdx}(?:_(\\d+))?$`);
  const thinking = new RegExp(`^th_${turnIdx}_(\\d+)$`);
  let assistantStep = 0;
  let thinkingStep = 0;
  let assistantRowUsed = false;
  for (const item of items) {
    if (item.type === "assistant_message") {
      const m = assistant.exec(item.id);
      if (m) {
        assistantRowUsed = true;
        assistantStep = Math.max(
          assistantStep,
          m[1] === undefined ? 1 : Number(m[1]) + 1,
        );
      }
    } else if (item.type === "thinking") {
      const m = thinking.exec(item.id);
      if (m) thinkingStep = Math.max(thinkingStep, Number(m[1]) + 1);
    }
  }
  return { turnIdx, assistantStep, thinkingStep, assistantRowUsed };
}
