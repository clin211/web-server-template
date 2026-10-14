import "@testing-library/jest-dom/vitest";
import { afterAll, afterEach, beforeAll } from "vitest";
import { fixtureServer } from "./fixture-server";

beforeAll(() => fixtureServer.listen({ onUnhandledRequest: "error" }));
afterEach(() => fixtureServer.resetHandlers());
afterAll(() => fixtureServer.close());
