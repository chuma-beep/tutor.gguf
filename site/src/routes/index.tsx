import { createRoute } from "@tanstack/react-router";
import { rootRoute } from "./root";
import shotBench from "@/assets/shot-bench.jpg";
import shotDesktop from "@/assets/shot-desktop.jpg";
import shotHero from "@/assets/shot-hero.jpg";
import shotTui from "@/assets/shot-tui.jpg";

const REPO = "https://github.com/chuma-beep/tutor.gguf";
const REPORT = `${REPO}/blob/main/REPORT.md`;
const MODEL = `${REPO}/releases/latest`;
const COMPLIANCE = `${REPO}/blob/main/COMPLIANCE.md`;
const QUESTIONS = `${REPO}/issues/new`;

const facts = [
  ["Throughput", "13.5 tok/s", "Audit profile · 16.6 at 4 threads"],
  ["Memory", "1.10 GB", "Peak RSS of a 7 GB budget"],
  ["Compute", "CPU only", "Integrated graphics"],
  ["Connectivity", "0 bytes", "Entirely offline"],
];

const stages = [
  ["01", "Input", "A mathematics problem is entered through the Bubble Tea terminal interface."],
  ["02", "Retrieve", "Chromem selects relevant worked examples from the local corpus."],
  ["03", "Infer", "Qwen2.5-Math-1.5B runs locally through llama.cpp in GGUF Q4_K_M."],
  ["04", "Render", "The response is formatted as readable, step-by-step mathematics."],
];

const plates = [
  [shotTui, "Plate 01", "Terminal interface", "Bubble Tea interface running the tutor entirely on-device."],
  [shotDesktop, "Plate 02", "Desktop application", "Wails desktop build: local retrieval and a worked derivative."],
  [shotBench, "Plate 03", "Performance study", "Measured throughput and memory on commodity hardware."],
];

function Index() {
  return (
    <div className="min-h-screen bg-muted text-foreground selection:bg-foreground selection:text-background">
      <header className="border-b border-border bg-background">
        <div className="mx-auto flex max-w-[1440px] items-center justify-between px-5 py-4 sm:px-8 lg:px-12">
          <a href="#top" className="text-sm font-semibold uppercase">tutor.gguf</a>
          <nav aria-label="Primary navigation" className="flex gap-5 font-mono text-[10px] uppercase text-muted-foreground">
            <a className="archive-link hidden sm:inline" href="#archive">Archive</a>
            <a className="archive-link hidden sm:inline" href="#specification">Specification</a>
            <a className="archive-link" href={REPO} target="_blank" rel="noreferrer">Repository ↗</a>
          </nav>
        </div>
      </header>

      <main id="top" className="mx-auto max-w-[1440px] px-4 py-4 sm:px-8 sm:py-8 lg:px-12 lg:py-12">
        <article className="border border-border bg-background">
          <header className="grid border-b border-border lg:grid-cols-12">
            <div className="p-7 sm:p-10 lg:col-span-8 lg:p-14">
              <p className="archive-label">Index / Project No. TG-001</p>
              <h1 className="mt-5 text-5xl font-medium uppercase leading-[0.9] sm:text-7xl lg:text-8xl">
                tutor.gguf<br /><span className="text-muted-foreground">Archive</span>
              </h1>
            </div>
            <dl className="grid grid-cols-2 border-t border-border font-mono text-[10px] uppercase lg:col-span-4 lg:border-l lg:border-t-0">
              {[["Year", "2026"], ["Location", "Anambra, NG"], ["Team", "chuma-beep"], ["Status", "Public / v0.2.0"]].map(([term, value]) => (
                <div key={term} className="border-b border-r border-border p-5 lg:last:border-b-0">
                  <dt className="text-muted-foreground">{term}</dt><dd className="mt-2 text-foreground">{value}</dd>
                </div>
              ))}
            </dl>
          </header>

          <section className="grid lg:grid-cols-12">
            <div className="space-y-10 border-b border-border p-7 sm:p-10 lg:col-span-4 lg:border-b-0 lg:border-r lg:p-12">
              <div>
                <p className="archive-label">Project abstract</p>
                <p className="mt-5 text-xl leading-snug">A fully offline mathematics tutor designed for Nigerian computer science undergraduates at distance-learning institutions.</p>
                <p className="mt-5 text-sm leading-7 text-muted-foreground">The system combines local retrieval with a compact quantized language model. It runs on an 8 GB laptop without a GPU, internet connection, cloud account, or per-token fee.</p>
              </div>
              <div className="border-t border-border pt-7">
                <p className="archive-label">System composition</p>
                <dl className="mt-5 space-y-3 font-mono text-[10px] uppercase">
                  {[["Model", "Qwen2.5-Math-1.5B"], ["Format", "GGUF Q4_K_M"], ["Runtime", "llama.cpp"], ["Retrieval", "chromem"], ["Interface", "Go / Bubble Tea"]].map(([term, value]) => (
                    <div key={term} className="flex justify-between gap-4 border-b border-border pb-2"><dt className="text-muted-foreground">{term}</dt><dd className="text-right">{value}</dd></div>
                  ))}
                </dl>
              </div>
              <a href={REPORT} target="_blank" rel="noreferrer" className="archive-action">View full report ↗</a>
            </div>

            <div className="bg-muted p-4 sm:p-8 lg:col-span-8 lg:p-12">
              <figure>
                <div className="aspect-[16/10] overflow-hidden border border-border bg-secondary">
                  <img src={shotHero} alt="The tutor.gguf terminal interface" className="h-full w-full object-cover grayscale transition duration-500 hover:grayscale-0" />
                </div>
                <figcaption className="mt-3 flex justify-between gap-4 font-mono text-[10px] uppercase text-muted-foreground">
                  <span>Fig. 01 / Primary interface</span><span>On-device operation</span>
                </figcaption>
              </figure>
              <div className="mt-10 grid grid-cols-2 gap-x-5 gap-y-8 sm:grid-cols-4">
                {facts.map(([label, value, note]) => (
                  <div key={label} className="border-t border-border pt-3">
                    <p className="archive-label text-muted-foreground">{label}</p>
                    <p className="mt-3 text-xl font-medium">{value}</p>
                    <p className="mt-1 text-xs text-muted-foreground">{note}</p>
                  </div>
                ))}
              </div>
            </div>
          </section>

          <section id="specification" className="archive-section">
            <div className="archive-section-head"><span>02</span><h2>System plan</h2><span>Local pipeline / four stages</span></div>
            <div className="grid md:grid-cols-4">
              {stages.map(([number, title, text], index) => (
                <article key={number} className="border-b border-border p-6 last:border-b-0 md:border-b-0 md:border-r md:last:border-r-0">
                  <div className="flex items-center gap-3 font-mono text-[10px] text-muted-foreground"><span>{number}</span><span className="h-px flex-1 bg-border" /><span>1:0{index + 1}</span></div>
                  <div className="my-8 flex h-24 items-center justify-center border border-border bg-muted" aria-hidden="true">
                    <span className="block border border-foreground" style={{ width: `${32 + index * 12}%`, height: `${32 + index * 10}%` }} />
                  </div>
                  <h3 className="text-lg font-medium">{title}</h3>
                  <p className="mt-3 text-sm leading-6 text-muted-foreground">{text}</p>
                </article>
              ))}
            </div>
          </section>

          <section id="archive" className="archive-section">
            <div className="archive-section-head"><span>03</span><h2>Drawing archive</h2><span>Interface / output / measurement</span></div>
            <div className="grid gap-px bg-border md:grid-cols-2">
              {plates.map(([src, number, title, caption], index) => (
                <figure key={number} className={`bg-background p-5 sm:p-8 ${index === 0 ? "md:col-span-2" : ""}`}>
                  <div className={`${index === 0 ? "aspect-[16/8]" : "aspect-[4/3]"} overflow-hidden border border-border bg-secondary`}>
                    <img src={src} alt={caption} loading="lazy" className="h-full w-full object-cover grayscale transition duration-500 hover:grayscale-0" />
                  </div>
                  <figcaption className="mt-4 grid gap-2 border-t border-border pt-3 sm:grid-cols-[8rem_1fr]">
                    <span className="archive-label text-muted-foreground">{number}</span>
                    <p className="text-sm"><strong className="font-medium">{title}.</strong> <span className="text-muted-foreground">{caption}</span></p>
                  </figcaption>
                </figure>
              ))}
            </div>
          </section>

          <section className="archive-section grid lg:grid-cols-12">
            <div className="border-b border-border p-7 sm:p-10 lg:col-span-5 lg:border-b-0 lg:border-r lg:p-12">
              <p className="archive-label">04 / Project record</p>
              <h2 className="mt-6 text-3xl font-medium">Built for useful inference on the hardware students already own.</h2>
              <p className="mt-6 max-w-lg text-sm leading-7 text-muted-foreground">Created by Wisdom Anwaegbu under the team name chuma-beep for the Math &amp; Scientific Reasoning track of the Africa Deep Tech Challenge 2026.</p>
            </div>
            <div className="grid sm:grid-cols-2 lg:col-span-7">
              {[["Source repository", "Implementation and documentation", REPO], ["Technical report", "Methodology and benchmark notes", REPORT], ["Model files", "Latest release artifacts", MODEL], ["Compliance record", "Competition alignment record", COMPLIANCE]].map(([label, note, href]) => (
                <a key={label} href={href} target="_blank" rel="noreferrer" className="group border-b border-r border-border p-7 transition-colors hover:bg-foreground hover:text-background">
                  <span className="archive-label text-muted-foreground group-hover:text-background">Resource</span>
                  <strong className="mt-8 block text-lg font-medium">{label} ↗</strong>
                  <span className="mt-2 block text-sm text-muted-foreground group-hover:text-background">{note}</span>
                </a>
              ))}
            </div>
          </section>

          <footer className="flex flex-col justify-between gap-5 border-t border-border bg-muted px-7 py-6 font-mono text-[10px] uppercase text-muted-foreground sm:flex-row sm:items-center">
            <span>Archive reference / TG-ADTC-2026-001</span>
            <a href={QUESTIONS} target="_blank" rel="noreferrer" className="archive-action">Questions &amp; feedback ↗</a>
            <span>GPL-3.0 / Public release</span>
          </footer>
        </article>
      </main>
    </div>
  );
}

export const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: Index,
});
