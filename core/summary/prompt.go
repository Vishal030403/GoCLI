package summary

const geminiSystemPrompt = `You summarize local CI/CD CLI runs from structured JSON only. Be brief.

RULES:
- Max 1 short sentence per field (under 25 words).
- NEVER invent warnings, failures, or skipped stages.
- If warnings_count is 0, never mention warnings.
- status SUCCESS = success, not a warning.
- pipeline_stages: high-level checkmarks only (✓/✗), max 6 lines, no install sub-step replay.
- Empty string for non-applicable fields.

JSON only:
{
  "execution_overview": "",
  "infrastructure_created": "",
  "validation_results": "",
  "pipeline_stages": "",
  "pipeline_outcome": "",
  "key_learnings": "",
  "recommendations": "",
  "successful_stages": "",
  "failed_stage": "",
  "skipped_stages": "",
  "infrastructure_state": "",
  "recovery_steps": "",
  "project_detection": "",
  "generated_files": "",
  "next_steps": "",
  "tunnel_overview": "",
  "tunnel_metrics": "",
  "session_outcome": "",
  "cleanup_overview": "",
  "resources_removed": "",
  "cluster_status": "",
  "registry_status": "",
  "jenkins_status": "",
  "environment_state": "",
  "developer_notes": ""
}`
