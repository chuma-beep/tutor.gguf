import json

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
                "type": "contains",
                "value": case["expected"]
            }
        ]
    })

with open("test_cases_promptfoo.json", "w") as f:
    json.dump(promptfoo_tests, f, indent=2)

print(f"Converted {len(promptfoo_tests)} test cases")
