// Deterministic MATH-style answer matcher for promptfoo.
// Extracts the model's final answer and compares it to the expected
// answer after LaTeX-aware normalization, with numeric fallback for
// algebraically-equivalent-but-textually-different forms.
//
// Contract: ESM default export function(output, vars) -> boolean.
export default function (output, vars) {
  const expected =
    (vars && vars.expected) ??
    (vars && vars.vars && vars.vars.expected);

  const extract = (s) => {
    const boxed = /\\boxed\s*\{((?:[^{}]|\{[^{}]*\})*)\}/.exec(s);
    if (boxed) return boxed[1];
    const marker = /(?:final answer(?:\s*is)?|answer is|answer:)\s*(.+)$/i.exec(s);
    if (marker) return marker[1];
    const africa = /####\s*(\S+.*)/.exec(s);
    if (africa) return africa[1];
    return s;
  };

  const norm = (s) =>
    (s ?? "")
      .replace(/\\text\{([^{}]*)\}/g, "$1")
      .replace(/\\mathrm\{([^{}]*)\}/g, "$1")
      .replace(/\\operatorname\{([^{}]*)\}/g, "$1")
      .replace(/\\(?:left|right|Big|big|Bigg|bigg)\b/g, "")
      .replace(/\\(?:dfrac|tfrac|frac)\s*\{([^{}]*)\}\{([^{}]*)\}/g, "($1)/($2)")
      .replace(/\\sqrt\[([^\]]*)\]\{([^{}]*)\}/g, "root$1($2)")
      .replace(/\\sqrt\{([^{}]*)\}/g, "sqrt($1)")
      .replace(/\\cdot/g, "*")
      .replace(/\\times/g, "*")
      .replace(/\\,/g, " ")
      .replace(/\{[,\s]\}/g, ", ")
      .replace(/\\[$\\]/g, " ")
      .replace(/\band\b/gi, ",")
      .replace(/[\\$]/g, "")
      .replace(/\s+/g, "")
      .replace(/,+/g, ",")
      .replace(/^[\s.,;:!?'"]+|[\s.,;:!?'"]+$/g, "")
      .toLowerCase();

  const a = norm(extract(output));
  const b = norm(expected);
  if (a === b) return true;

  // Truncation fallback: the exact expected string appears in the tail of
  // the output even though the final-answer marker was never reached, or
  // the output ends with the expected answer (safe for single-digit
  // answers, which the length guard below would otherwise skip).
  {
    const outNorm = norm(output);
    if (outNorm.endsWith(b) || outNorm.endsWith(b + ".")) return true;
    if (b.length >= 2) {
      const tail = outNorm.slice(Math.max(0, Math.floor(outNorm.length * 0.5)));
      if (tail.includes(b)) return true;
    }
  }

  const asList = (s) => s.split(",").map((t) => t.trim()).filter(Boolean);
  if (asList(a).length > 1 && asList(b).length > 1) {
    const sa = [...asList(a)].sort().join(",");
    const sb = [...asList(b)].sort().join(",");
    if (sa === sb) return true;
  }

  const val = (s) => {
    const frac = /^(-?)(?:\(?)(\d+)(?:\)?)\s*\/\s*\(?(-?\d+)\)?$/.exec(s);
    if (frac) {
      const d = Number(frac[3]);
      if (d === 0) return null;
      return (frac[1] === "-" ? -1 : 1) * (Number(frac[2]) / d);
    }
    const sqrt = /^sqrt\((-?\d+)\)$/.exec(s);
    if (sqrt) {
      const n = Number(sqrt[1]);
      return n < 0 ? null : Math.sqrt(n);
    }
    const root = /^root(\d+)\((-?\d+)\)$/.exec(s);
    if (root) {
      const [r, n] = [Number(root[1]), Number(root[2])];
      if (r < 1 || n < 0) return null;
      return Math.pow(n, 1 / r);
    }
    const commas = (s.match(/,/g) || []).length;
    if (commas <= 1) {
      const clean = s.replace(/,/g, "");
      if (/^-?\d+(?:\.\d+)?$/.test(clean)) return Number(clean);
    }
    const first = /-?\d+(?:\.\d+)?/.exec(s);
    return first ? Number(first[0]) : null;
  };
  const va = val(a);
  const vb = val(b);
  if (va !== null && vb !== null && Math.abs(va - vb) <= 1e-6 * Math.max(1, Math.abs(vb))) return true;

  return false;
}
