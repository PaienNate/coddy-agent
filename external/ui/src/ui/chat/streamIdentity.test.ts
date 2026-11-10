import { expect, test } from "vitest";
import type { TranscriptItem } from "./types";
import { newTurnIdentity, resumedTurnIdentity } from "./streamIdentity";

const user = (turn: number): TranscriptItem => ({
  id: `u_${turn}`,
  type: "user_message",
  content: `question ${turn}`,
});

test("a new send opens the turn after the turns already on screen", () => {
  expect(newTurnIdentity([user(0), user(1)], 3)).toEqual({
    turnIdx: 6,
    assistantStep: 0,
    thinkingStep: 0,
  });
});

test("a rejoining reader continues the turn it is already showing", () => {
  const items: TranscriptItem[] = [
    user(2),
    { id: "th_2_0", type: "thinking", status: "completed", content: "look" },
    { id: "as_2", type: "assistant_message", content: "first" },
    { id: "as_2_1", type: "assistant_message", content: "second" },
  ];
  expect(resumedTurnIdentity(items, 1)).toEqual({
    turnIdx: 2,
    assistantStep: 2,
    thinkingStep: 1,
    assistantRowUsed: true,
  });
});

test("a wake or goal row opens a turn the same way a prompt does", () => {
  const items: TranscriptItem[] = [
    { id: "wake_0", type: "background_wake", tasks: [] },
    {
      id: "goal_1",
      type: "goal_turn",
      turn: {
        kind: "kickoff",
        index: 1,
        limit: 1,
        objective: "go",
        reason: "",
        remaining: [],
      },
    },
  ];
  expect(newTurnIdentity(items, 0).turnIdx).toBe(3);
});
