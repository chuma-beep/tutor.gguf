import json
import os
import random

random.seed(42)

# Sample from Hendrycks MATH
hendrycks_dir = "../data/raw/hendrycks_math/train"
files = os.listdir(hendrycks_dir)
sample_files = random.sample(files, min(30, len(files)))

test_cases = []
for fname in sample_files:
    with open(os.path.join(hendrycks_dir, fname)) as f:
        item = json.load(f)
    
    # Extract expected answer from inside \boxed{}
    solution = item["solution"]
    if "\\boxed{" in solution:
        start = solution.index("\\boxed{") + len("\\boxed{")
        depth = 1
        end = start
        while depth > 0 and end < len(solution):
            if solution[end] == "{":
                depth += 1
            elif solution[end] == "}":
                depth -= 1
            end += 1
        expected = solution[start:end-1]
    else:
        continue  # skip problems without a clean boxed answer

    test_cases.append({
        "problem": item["problem"],
        "expected": expected,
        "type": item["type"],
        "level": item["level"]
    })

with open("test_cases.json", "w") as f:
    json.dump(test_cases, f, indent=2)

print(f"Generated {len(test_cases)} test cases")
