import json

JUDGE_PROVIDER = {
    "id": "openai:chat:qwen2.5-3b-instruct",
    "config": {
        "apiBaseUrl": "http://localhost:8083/v1",
        "apiKey": "none",
        "response_format": {
            "type": "json_schema",
            "json_schema": {
                "name": "grade",
                "schema": {
                    "type": "object",
                    "properties": {
                        "reason": {"type": "string"},
                        "pass": {"type": "boolean"},
                        "score": {"type": "number"},
                    },
                    "required": ["pass", "reason", "score"],
                },
            },
        },
    },
}

RUBRIC = (
    "Judge whether the final answer in the model output is correct for the "
    'problem, given that the expected answer is "{{expected}}". '
    "Mathematically equivalent expressions (different ordering of terms, "
    "different but equivalent forms, fractions vs decimals, \"and\" vs "
    "commas in list answers) must count as correct. Ignore reasoning quality "
    "and formatting — only the final answer matters. "
    "Respond with exactly PASS or FAIL."
)

with open("test_cases.json") as f:
    raw = json.load(f)

promptfoo_tests = []
for case in raw:
    promptfoo_tests.append({
        "vars": {
            "problem": case["problem"],
            "expected": case["expected"]
        },
        "assert": [
            {
                "type": "javascript",
                "value": "file://./answer_assert.js"
            },
            {
                "type": "llm-rubric",
                "provider": JUDGE_PROVIDER,
                "value": RUBRIC
            }
        ]
    })

with open("test_cases_promptfoo.json", "w") as f:
    json.dump(promptfoo_tests, f, indent=2)

print(f"Converted {len(promptfoo_tests)} test cases")
