import { PHASE_GRADIENTS, type PhaseKey } from "@/lib/solar";

const ORDER: PhaseKey[] = [
  "night",
  "morningBlue",
  "morningGolden",
  "day",
  "eveningGolden",
  "eveningBlue",
];

export default function PhaseBackground({ phase }: { phase: PhaseKey }) {
  return (
    <div className="fixed inset-0 -z-10" aria-hidden>
      {ORDER.map((key) => (
        <div
          key={key}
          className="absolute inset-0 transition-opacity duration-1000 ease-in-out"
          style={{
            backgroundImage: PHASE_GRADIENTS[key],
            opacity: key === phase ? 1 : 0,
          }}
        />
      ))}
    </div>
  );
}
