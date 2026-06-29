"use client";

import { cn } from "@/lib/utils";
import type { HTMLAttributes,MouseEventHandler } from "react";
import { useEffect,useId,useRef,useState } from "react";

type MermaidContentProps = HTMLAttributes<HTMLElement> & {
  as?: "article" | "div";
  html: string;
};

type MermaidColorScheme = "dark" | "light";

type MermaidRenderer = (typeof import("mermaid"))["default"];

let mermaidModulePromise: Promise<MermaidRenderer> | null = null;
let mermaidRenderQueue: Promise<void> = Promise.resolve();

export function MermaidContent({
  as: Component = "div",
  className,
  html,
  onClickCapture,
  ...props
}: MermaidContentProps) {
  const rootRef = useRef<HTMLElement>(null);
  const setRootRef = (node: HTMLElement | null) => {
    rootRef.current = node;
  };
  const instanceId = useId().replace(/[^a-zA-Z0-9_-]/g, "");
  const renderPassRef = useRef(0);
  const [renderTick, setRenderTick] = useState(0);
  const [colorScheme, setColorScheme] = useState<MermaidColorScheme>("dark");

  useEffect(() => {
    const updateColorScheme = () => {
      setColorScheme(readMermaidColorScheme());
    };

    updateColorScheme();

    const observer = new MutationObserver(updateColorScheme);
    observer.observe(document.documentElement, {
      attributeFilter: ["data-yuem-theme"],
      attributes: true,
    });

    const colorSchemeQuery = window.matchMedia("(prefers-color-scheme: light)");
    colorSchemeQuery.addEventListener("change", updateColorScheme);

    return () => {
      observer.disconnect();
      colorSchemeQuery.removeEventListener("change", updateColorScheme);
    };
  }, []);

  useEffect(() => {
    const rootElement = rootRef.current;
    if (!rootElement) {
      return;
    }
    const root = rootElement;

    let cancelled = false;
    renderPassRef.current += 1;
    const renderPass = renderPassRef.current;

    async function renderDiagrams() {
      const diagrams = Array.from(root.querySelectorAll<HTMLElement>("[data-mermaid-source]"));
      if (diagrams.length === 0) {
        return;
      }

      const diagramSources = diagrams.map((diagram) => (
        diagram.getAttribute("data-mermaid-source")
        || diagram.textContent
        || ""
      ).trim());
      diagrams.forEach((diagram, index) => {
        if (diagramSources[index]) {
          diagram.setAttribute("data-mermaid-status", "loading");
        }
      });

      await enqueueMermaidRender(async () => {
        if (cancelled) {
          return;
        }

        let mermaid: MermaidRenderer;
        try {
          mermaid = await loadMermaid();
          mermaid.initialize(getMermaidConfig(colorScheme));
        } catch {
          if (!cancelled) {
            markMermaidErrors(diagrams, diagramSources);
          }
          return;
        }

        for (const [index, diagram] of diagrams.entries()) {
          const source = diagramSources[index];
          if (!source || cancelled) {
            continue;
          }

          try {
            const id = `mermaid-${instanceId}-${colorScheme}-${renderPass}-${renderTick}-${index}`;
            const { svg } = await mermaid.render(id, source);
            if (cancelled) {
              return;
            }
            diagram.innerHTML = svg;
            diagram.setAttribute("data-mermaid-status", "rendered");
          } catch {
            if (!cancelled) {
              diagram.textContent = source;
              diagram.setAttribute("data-mermaid-status", "error");
            }
          }
        }
      });
    }

    void renderDiagrams();

    return () => {
      cancelled = true;
    };
  }, [colorScheme, html, instanceId, renderTick]);

  const handleClickCapture: MouseEventHandler<HTMLElement> = (event) => {
    onClickCapture?.(event);
    if (event.defaultPrevented) {
      return;
    }

    const target = event.target;
    if (target instanceof Element && target.closest("[data-mermaid-status='error']")) {
      setRenderTick((tick) => tick + 1);
    }
  };

  return (
    <Component
      ref={setRootRef}
      className={cn("mermaid-render-scope", className)}
      onClickCapture={handleClickCapture}
      dangerouslySetInnerHTML={{ __html: html }}
      {...props}
    />
  );
}

function loadMermaid() {
  mermaidModulePromise ??= import("mermaid")
    .then((module) => module.default)
    .catch((error) => {
      mermaidModulePromise = null;
      throw error;
    });
  return mermaidModulePromise;
}

function enqueueMermaidRender(render: () => Promise<void>) {
  const queuedRender = mermaidRenderQueue.then(render, render);
  mermaidRenderQueue = queuedRender.catch(() => undefined);
  return queuedRender;
}

function markMermaidErrors(diagrams: HTMLElement[], sources: string[]) {
  diagrams.forEach((diagram, index) => {
    const source = sources[index];
    if (!source) {
      return;
    }
    diagram.textContent = source;
    diagram.setAttribute("data-mermaid-status", "error");
  });
}

function readMermaidColorScheme(): MermaidColorScheme {
  const explicitTheme = document.documentElement.dataset.yuemTheme;
  if (explicitTheme === "light" || explicitTheme === "dark") {
    return explicitTheme;
  }
  return window.matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark";
}

function getMermaidConfig(colorScheme: MermaidColorScheme) {
  const light = colorScheme === "light";
  return {
    securityLevel: "strict" as const,
    startOnLoad: false,
    theme: light ? "base" as const : "dark" as const,
    themeVariables: light
      ? {
          background: "transparent",
          darkMode: false,
          fontFamily: "inherit",
          primaryColor: "#fff1f2",
          primaryBorderColor: "#ff8aa0",
          primaryTextColor: "#25252b",
          secondaryColor: "#ffffff",
          secondaryBorderColor: "rgba(28, 28, 32, 0.18)",
          secondaryTextColor: "#25252b",
          tertiaryColor: "#f6f6f7",
          tertiaryBorderColor: "rgba(28, 28, 32, 0.14)",
          tertiaryTextColor: "#25252b",
          lineColor: "#ff2442",
          textColor: "#25252b",
        }
      : {
          background: "transparent",
          darkMode: true,
          fontFamily: "inherit",
          primaryColor: "#25252b",
          primaryTextColor: "#f8fafc",
          lineColor: "#f472b6",
          textColor: "#f8fafc",
        },
  };
}
