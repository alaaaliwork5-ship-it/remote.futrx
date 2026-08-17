import { requestJson } from "./apiRequest";
import { API_ROUTES } from "../config/routes";

export interface ApprovalDecision {
  decision: "allow" | "deny";
}

export const approvalApi = {
  decide: (approvalId: string, decision: "allow" | "deny") =>
    requestJson<ApprovalDecision>(
      "POST",
      API_ROUTES.approvals.decision(approvalId),
      { decision }
    ),
};
