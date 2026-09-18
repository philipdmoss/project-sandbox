const UNITARIAN_UNIVERSALIST_PRINCIPLES = [
  "The inherent worth and dignity of every person",
  "Justice, equity and compassion in human relations",
  "Acceptance of one another and encouragement to spiritual growth in our congregations",
  "A free and responsible search for truth and meaning",
  "The right of conscience and the use of the democratic process within our congregations and in society at large",
  "The goal of world community with peace, liberty, and justice for all",
  "Respect for the interdependent web of all existence of which we are a part",
];

export async function GET() {
  const principle =
    UNITARIAN_UNIVERSALIST_PRINCIPLES[
      Math.floor(Math.random() * UNITARIAN_UNIVERSALIST_PRINCIPLES.length)
    ];
  return new Response(`Yes, I'm here.\n\n${principle}\n`, {
    headers: { "content-type": "text/plain; charset=utf-8" },
  });
}
