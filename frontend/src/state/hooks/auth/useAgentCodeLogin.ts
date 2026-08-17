import { useEffect, useRef, useState } from "preact/hooks";
import type { ClaudeLoginPhase } from "../../../models/auth";
import { agentAuthApi } from "../../../api/agents/auth/agentAuthApi";

// Generic authorization-code login flow (the Claude pattern): the CLI emits an
// OAuth URL and expects a code pasted back. Driven entirely by the agent
// catalog id so any code-flow provider reuses it.
export function useAgentCodeLogin(agentId: string, onDone: () => void) {
  const [phase, setPhaseState] = useState<ClaudeLoginPhase>("idle");
  const [authUrl, setAuthUrl] = useState("");
  const [code, setCode] = useState("");
  const [errorMessage, setErrorMessage] = useState("");
  const phaseRef = useRef<ClaudeLoginPhase>("idle");
  const api = agentAuthApi(agentId);

  function setPhase(next: ClaudeLoginPhase) {
    phaseRef.current = next;
    setPhaseState(next);
  }

  useEffect(() => {
    return () => {
      if (phaseRef.current === "starting" || phaseRef.current === "awaiting-code") {
        api.cancelLogin().catch(() => {});
      }
    };
  }, [api]);

  async function startLogin() {
    setPhase("starting");
    setErrorMessage("");
    try {
      const response = await api.startLogin();
      setAuthUrl(response.url);
      setPhase("awaiting-code");
    } catch (error) {
      setErrorMessage((error as Error).message);
      setPhase("error");
    }
  }

  async function submitCode() {
    const trimmed = code.trim();
    if (!trimmed) return;
    setPhase("submitting");
    setErrorMessage("");
    try {
      await api.submitCode(trimmed);
      setPhase("done");
      setTimeout(onDone, 700);
    } catch (error) {
      setErrorMessage((error as Error).message);
      setPhase("error");
    }
  }

  async function cancel() {
    try {
      await api.cancelLogin();
    } catch {}
    reset();
  }

  function reset() {
    setPhase("idle");
    setCode("");
    setAuthUrl("");
    setErrorMessage("");
  }

  return {
    phase,
    authUrl,
    code,
    setCode,
    errorMessage,
    startLogin,
    submitCode,
    cancel,
    reset,
  };
}
